resource "wayfinder_repositories" "nvidia-device-plugin" {
  metadata {
    name = "nvidia-device-plugin"
  }

  spec {
    url      = "https://nvidia.github.io/k8s-device-plugin"
    interval = "60m"
    suspend  = false
    timeout  = "60s"

    selectors {
      match_expressions {
        key      = "nvidia.com/gpu"
        operator = "Exists"
      }
      match_expressions {
        key      = "appvia.io/provider"
        operator = "In"
        values   = ["EKS"]
      }
    }
  }
}

resource "wayfinder_packages" "nvidia-device-plugin" {
  metadata {
    name = "nvidia-device-plugin"
  }

  spec {
    summary           = "NVIDIA device plugin for Kubernetes"
    install_namespace = "kube-system"
    name              = "Kubernetes nVidia Device Plugin"
    release_name      = "nvidia-device-plugin"
    version           = "v0.14.0"

    selectors {
      match_expressions {
        key      = "appvia.io/managementcluster"
        operator = "DoesNotExist"
      }
      match_expressions {
        key      = "nvidia.com/gpu"
        operator = "Exists"
      }
      match_expressions {
        key      = "appvia.io/provider"
        operator = "In"
        values   = ["EKS"]
      }
    }

    source {
      helm {
        name    = "nvidia-device-plugin"
        url     = "https://nvidia.github.io/k8s-device-plugin"
        version = "v0.14.0"
      }
    }

    values = <<EOF
      nodeSelector"
        "nvidia.com/gpu": "true"
EOF
  }
}
