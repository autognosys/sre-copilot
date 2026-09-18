output "k3s_node_name" {
  value = proxmox_virtual_environment_vm.k3s_node.name
}

output "k3s_node_ip" {
  description = "Static IP of the k3s node (CIDR stripped) — feed this into the Ansible inventory."
  value       = split("/", var.vm_ip_address)[0]
}

output "k3s_node_ssh_user" {
  value = var.vm_username
}

output "ssh_command" {
  description = "How to reach the node — via pve1 as a jump host, since it has no public IP."
  value       = "ssh -J root@${split(":", split("/", var.pve_api_endpoint)[2])[0]} ${var.vm_username}@${split("/", var.vm_ip_address)[0]}"
}
