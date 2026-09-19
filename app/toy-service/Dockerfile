FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
# Pins in requirements.txt are a reasonable starting point, not guaranteed
# mutually compatible on the day you build this — if `pip install` fails
# on a version conflict, relax the pins and re-lock.
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

ENV OTEL_SERVICE_NAME=order-service
# Overridden per-environment via Deployment env — this is just a placeholder
# default for local `docker run` testing.
ENV OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

EXPOSE 8000

# Tracing is set up explicitly in main.py (TracerProvider + FastAPIInstrumentor)
# rather than via the `opentelemetry-instrument` auto-instrumentation wrapper —
# the auto wrapper silently produced zero spans in testing. Plain entrypoint.
CMD ["python", "main.py"]
