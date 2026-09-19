"""
order-service — the toy workload for sre-copilot's "bad deploy" scenario.

Normal mode: a boring FastAPI app with a couple of endpoints, emitting
traces via OTel auto-instrumentation to whatever OTLP endpoint is set in
OTEL_EXPORTER_OTLP_ENDPOINT.

"Bad deploy" mode: set LEAK_RATE_MB_PER_SEC to a nonzero value (via the
Deployment's env, i.e. a "deploy") and the service will steadily allocate
memory in a background thread until it hits the container's memory limit
and gets OOMKilled — a deterministic, reproducible incident to feed the
triage agent.

CRASH_ON_STARTUP=true is the other failure mode: the process exits
immediately, producing a CrashLoopBackOff instead of an OOMKill — useful
for testing that the triage agent distinguishes the two root-cause
categories rather than just pattern-matching "pod is unhealthy".
"""

import logging
import os
import sys
import threading
import time

from fastapi import FastAPI
import uvicorn

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
    stream=sys.stdout,
)
log = logging.getLogger("order-service")

# Explicit OTel SDK setup rather than relying on the `opentelemetry-instrument`
# auto-instrumentation wrapper — auto-instrumentation silently produced zero
# spans in testing (FastAPI's instrumentor entry point never fired, no error
# logged either), which is exactly the kind of invisible failure you don't
# want in a reference app meant to be legible. Explicit is more code but
# fails loudly if it fails at all.
OTLP_ENDPOINT = os.getenv(
    "OTEL_EXPORTER_OTLP_ENDPOINT",
    "http://localhost:4317",
)
SERVICE_NAME = os.getenv("OTEL_SERVICE_NAME", "order-service")

trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({"service.name": SERVICE_NAME}))
)
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(OTLPSpanExporter(endpoint=OTLP_ENDPOINT, insecure=True))
)

CRASH_ON_STARTUP = os.getenv("CRASH_ON_STARTUP", "false").lower() == "true"
LEAK_RATE_MB_PER_SEC = float(os.getenv("LEAK_RATE_MB_PER_SEC", "0"))

if CRASH_ON_STARTUP:
    log.error("CRASH_ON_STARTUP is set — exiting immediately to simulate a bad deploy")
    sys.exit(1)

app = FastAPI(title="order-service")
FastAPIInstrumentor.instrument_app(app)

_leak_buffer: list[bytes] = []


def _leak_memory():
    """Background thread: allocates LEAK_RATE_MB_PER_SEC of memory every
    second and never frees it. Only runs if LEAK_RATE_MB_PER_SEC > 0."""
    chunk_bytes = int(LEAK_RATE_MB_PER_SEC * 1024 * 1024)
    while True:
        _leak_buffer.append(bytes(chunk_bytes))
        log.info(
            "leaked another %.1fMB (approx total held: %.1fMB)",
            LEAK_RATE_MB_PER_SEC,
            len(_leak_buffer) * LEAK_RATE_MB_PER_SEC,
        )
        time.sleep(1)


if LEAK_RATE_MB_PER_SEC > 0:
    log.warning(
        "LEAK_RATE_MB_PER_SEC=%.1f — this build will steadily consume memory "
        "until OOMKilled. This is the injected 'bad deploy' condition.",
        LEAK_RATE_MB_PER_SEC,
    )
    threading.Thread(target=_leak_memory, daemon=True).start()


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.get("/orders/{order_id}")
def get_order(order_id: str):
    log.info("fetching order %s", order_id)
    return {"order_id": order_id, "status": "processing"}


@app.get("/")
def root():
    return {"service": "order-service", "leak_mode": LEAK_RATE_MB_PER_SEC > 0}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
