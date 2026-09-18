# Repo structure (for first commit)

```
sre-copilot/
├── README.md                  ← public entry point: what it is, quickstart, link to ARCHITECTURE.md
├── ARCHITECTURE.md            ← the doc drafted above
├── LICENSE                    ← pick one (MIT or Apache-2.0 are the common defaults for this kind of project)
├── .gitignore                 ← as drafted above
│
├── infra/
│   ├── terraform/              ← Proxmox VM provisioning
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── terraform.tfvars.example   ← committed template, real .tfvars is gitignored
│   └── ansible/                ← k3s bootstrap, OTel + OpenObserve deployment
│       ├── playbook.yml
│       ├── inventory.example.ini
│       └── roles/
│
├── agent/                       ← the triage agent itself
│   ├── src/
│   ├── config.example.yaml      ← real config.yaml is gitignored
│   └── requirements.txt
│
├── fault-injection/              ← the manifests/scripts that simulate the bad-deploy incident class
│   └── scenarios/
│       ├── bad-image-tag.yaml
│       ├── broken-healthcheck.yaml
│       └── tight-memory-limit.yaml
│
├── eval/                          ← the fixed eval suite + published results
│   ├── cases/
│   ├── run_eval.py
│   └── RESULTS.md                 ← honest pass/fail, including the case it doesn't solve
│
└── docs/
    └── diagrams/                  ← architecture diagram source + exported image
```

## Notes

- **`config.example.yaml` / `terraform.tfvars.example` pattern**: commit the template with placeholder values, gitignore the real file. This is the standard way to show *what* config is needed without leaking *your* config.
- **`eval/RESULTS.md`** is worth treating as a first-class document, not an afterthought — it's the part that makes this credible rather than just another demo repo. Publish it even when (especially when) it shows the one case the agent gets wrong or punts on.
- **LICENSE**: worth picking before the first public push rather than after — MIT is the simplest if you want maximum reuse with attribution; Apache-2.0 if you want explicit patent-grant language. Either is a fine default for this kind of project.
- **No mention anywhere** of the business/client motivation, Skyler, kis.ai, or the market analysis — that context lives only in your private planning doc, not in the repo at any level (code comments, commit messages, or docs).
