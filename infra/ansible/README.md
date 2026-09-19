# infra/ansible

Installs k3s on the node Terraform created (`infra/terraform`) and fetches its
kubeconfig back to your machine.

## Run it

```bash
cd infra/ansible
ansible-playbook playbook.yml
```

## Why `kubectl` needs a tunnel

k3s's kubeconfig points at `https://127.0.0.1:6443` by default — correct on
the node itself, but symphony has no direct route to the node's private
address (`192.168.100.10`, on `vmbr1`, NAT-only, no public IP). So `kubectl`
from symphony needs an SSH tunnel through `pve1` first:

```bash
ssh -L 6443:192.168.100.10:6443 root@65.108.102.155 -N
```

Leave that running in its own terminal, then in another:

```bash
KUBECONFIG=infra/ansible/fetched/sre-copilot-kubeconfig.yaml kubectl get nodes
```

`fetched/sre-copilot-kubeconfig.yaml` is gitignored — it's a live cluster
credential, same handling as the Terraform token.
