resource "proxmox_virtual_environment_vm" "k3s_node" {
  name        = var.k3s_node_name
  node_name   = var.pve_node
  vm_id       = var.k3s_node_vm_id
  description = "Managed by Terraform — sre-copilot k3s node. Manual changes will be reverted."

  clone {
    vm_id = var.template_vm_id
    full  = true
  }

  cpu {
    cores = var.k3s_node_cores
    type  = "host"
  }

  memory {
    dedicated = var.k3s_node_memory
  }

  agent {
    enabled = true
  }

  disk {
    datastore_id = "local-zfs"
    interface    = "scsi0"
    size         = var.k3s_node_disk_size
  }

  network_device {
    bridge = var.network_bridge
  }

  initialization {
    datastore_id = "local-zfs"

    ip_config {
      ipv4 {
        address = var.vm_ip_address
        gateway = var.vm_gateway
      }
    }

    dns {
      servers = var.vm_dns_servers
    }

    user_account {
      username = var.vm_username
      keys     = [trimspace(file(pathexpand(var.ssh_public_key_path)))]
    }
  }

  operating_system {
    type = "l26" # Linux 2.6+ kernel ABI, correct for any modern Linux incl. Ubuntu 24.04
  }

  lifecycle {
    ignore_changes = [
      network_device, # avoid MAC-address drift triggering unwanted recreation
    ]
  }
}
