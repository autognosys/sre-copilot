variable "pve_api_endpoint" {
  description = "Proxmox VE API endpoint, e.g. https://65.108.102.155:8006/api2/json"
  type        = string
}

variable "pve_api_token" {
  description = "Proxmox API token in the form 'user@realm!token-id=secret'. Set via TF_VAR_pve_api_token or terraform.tfvars (gitignored) — never commit this."
  type        = string
  sensitive   = true
}

variable "pve_tls_insecure" {
  description = "Skip TLS verification for the PVE API (true for self-signed certs, common on a single-node homelab-style host)."
  type        = bool
  default     = true
}

variable "pve_node" {
  description = "Proxmox node name to provision on."
  type        = string
  default     = "pve1"
}

variable "template_vm_id" {
  description = "VMID of the cloud-init-ready template to clone."
  type        = number
  default     = 9000
}

variable "k3s_node_name" {
  description = "Hostname / VM name for the k3s node."
  type        = string
  default     = "sre-copilot-k3s-1"
}

variable "k3s_node_vm_id" {
  description = "VMID to assign to the cloned k3s node."
  type        = number
  default     = 9001
}

variable "k3s_node_cores" {
  description = "vCPU count for the k3s node."
  type        = number
  default     = 4
}

variable "k3s_node_memory" {
  description = "Memory (MB) for the k3s node."
  type        = number
  default     = 16384
}

variable "k3s_node_disk_size" {
  description = "Disk size (GB) for the k3s node."
  type        = number
  default     = 100
}

variable "network_bridge" {
  description = "Proxmox bridge the VM's NIC attaches to (private NAT bridge, not the public-facing one)."
  type        = string
  default     = "vmbr1"
}

variable "vm_ip_address" {
  description = "Static IP (CIDR) for the k3s node on the private bridge."
  type        = string
  default     = "192.168.100.10/24"
}

variable "vm_gateway" {
  description = "Gateway for the k3s node, i.e. the host's address on the private bridge."
  type        = string
  default     = "192.168.100.1"
}

variable "vm_dns_servers" {
  description = "DNS servers for the VM (public resolvers, since there's no internal DNS yet)."
  type        = list(string)
  default     = ["1.1.1.1", "8.8.8.8"]
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key to inject via cloud-init."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "vm_username" {
  description = "Default cloud-init user created on the VM."
  type        = string
  default     = "ubuntu"
}
