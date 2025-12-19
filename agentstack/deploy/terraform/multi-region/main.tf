# Multi-Region Infrastructure for AgentStack
# Sovereign AI Platform for Europe

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "gcs" {
    bucket = "agentstack-terraform-state"
    prefix = "multi-region"
  }
}

# Variables
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "environment" {
  description = "Environment (production, staging, development)"
  type        = string
  default     = "production"
}

variable "regions" {
  description = "List of European regions for deployment"
  type = list(object({
    name     = string
    location = string
    zone     = string
    primary  = bool
  }))
  default = [
    {
      name     = "europe-west1"
      location = "europe-west1"
      zone     = "europe-west1-b"
      primary  = true
    },
    {
      name     = "europe-west4"
      location = "europe-west4"
      zone     = "europe-west4-a"
      primary  = false
    },
    {
      name     = "europe-north1"
      location = "europe-north1"
      zone     = "europe-north1-a"
      primary  = false
    }
  ]
}

variable "gke_config" {
  description = "GKE cluster configuration"
  type = object({
    min_node_count     = number
    max_node_count     = number
    machine_type       = string
    disk_size_gb       = number
    preemptible        = bool
  })
  default = {
    min_node_count = 3
    max_node_count = 10
    machine_type   = "e2-standard-4"
    disk_size_gb   = 100
    preemptible    = false
  }
}

variable "database_config" {
  description = "Cloud SQL configuration"
  type = object({
    tier                     = string
    disk_size_gb             = number
    disk_autoresize          = bool
    availability_type        = string
    backup_enabled           = bool
    point_in_time_recovery   = bool
  })
  default = {
    tier                   = "db-custom-4-16384"
    disk_size_gb           = 100
    disk_autoresize        = true
    availability_type      = "REGIONAL"
    backup_enabled         = true
    point_in_time_recovery = true
  }
}

variable "redis_config" {
  description = "Memorystore Redis configuration"
  type = object({
    tier          = string
    memory_size_gb = number
    replica_count  = number
  })
  default = {
    tier           = "STANDARD_HA"
    memory_size_gb = 5
    replica_count  = 2
  }
}

# Local values
locals {
  primary_region = [for r in var.regions : r if r.primary][0]
  labels = {
    environment = var.environment
    managed_by  = "terraform"
    project     = "agentstack"
    compliance  = "gdpr"
  }
}

# Enable required APIs
resource "google_project_service" "apis" {
  for_each = toset([
    "container.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "servicenetworking.googleapis.com",
    "secretmanager.googleapis.com",
    "monitoring.googleapis.com",
    "logging.googleapis.com",
    "cloudtrace.googleapis.com",
    "compute.googleapis.com",
  ])

  project = var.project_id
  service = each.value

  disable_on_destroy = false
}

# VPC Network
resource "google_compute_network" "main" {
  name                    = "agentstack-network"
  project                 = var.project_id
  auto_create_subnetworks = false

  depends_on = [google_project_service.apis]
}

# Subnetworks for each region
resource "google_compute_subnetwork" "regional" {
  for_each = { for r in var.regions : r.name => r }

  name          = "agentstack-subnet-${each.value.name}"
  project       = var.project_id
  network       = google_compute_network.main.id
  region        = each.value.location
  ip_cidr_range = "10.${index(var.regions, each.value) + 1}.0.0/20"

  secondary_ip_range {
    range_name    = "pods"
    ip_cidr_range = "10.${index(var.regions, each.value) + 1}.16.0/20"
  }

  secondary_ip_range {
    range_name    = "services"
    ip_cidr_range = "10.${index(var.regions, each.value) + 1}.32.0/20"
  }

  private_ip_google_access = true

  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling        = 0.5
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

# Private service connection for Cloud SQL
resource "google_compute_global_address" "private_ip" {
  name          = "agentstack-private-ip"
  project       = var.project_id
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.main.id
}

resource "google_service_networking_connection" "private_vpc" {
  network                 = google_compute_network.main.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip.name]
}

# GKE Clusters per region
resource "google_container_cluster" "regional" {
  for_each = { for r in var.regions : r.name => r }

  name     = "agentstack-gke-${each.value.name}"
  project  = var.project_id
  location = each.value.location

  # Remove default node pool
  remove_default_node_pool = true
  initial_node_count       = 1

  network    = google_compute_network.main.id
  subnetwork = google_compute_subnetwork.regional[each.key].id

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }

  # Workload Identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Network policy
  network_policy {
    enabled  = true
    provider = "CALICO"
  }

  # Private cluster
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.${index(var.regions, each.value)}.0/28"
  }

  # Master authorized networks
  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "0.0.0.0/0"
      display_name = "All (should be restricted in production)"
    }
  }

  # Addons
  addons_config {
    http_load_balancing {
      disabled = false
    }
    horizontal_pod_autoscaling {
      disabled = false
    }
    network_policy_config {
      disabled = false
    }
    gce_persistent_disk_csi_driver_config {
      enabled = true
    }
  }

  # Logging and monitoring
  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS"]
    managed_prometheus {
      enabled = true
    }
  }

  # Maintenance window
  maintenance_policy {
    recurring_window {
      start_time = "2024-01-01T02:00:00Z"
      end_time   = "2024-01-01T06:00:00Z"
      recurrence = "FREQ=WEEKLY;BYDAY=SU"
    }
  }

  resource_labels = local.labels

  depends_on = [google_project_service.apis]
}

# Node pools for each cluster
resource "google_container_node_pool" "primary" {
  for_each = { for r in var.regions : r.name => r }

  name       = "primary-pool"
  project    = var.project_id
  cluster    = google_container_cluster.regional[each.key].name
  location   = each.value.location
  node_count = var.gke_config.min_node_count

  autoscaling {
    min_node_count = var.gke_config.min_node_count
    max_node_count = var.gke_config.max_node_count
  }

  node_config {
    machine_type = var.gke_config.machine_type
    disk_size_gb = var.gke_config.disk_size_gb
    disk_type    = "pd-ssd"
    preemptible  = var.gke_config.preemptible

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    labels = local.labels
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  upgrade_settings {
    max_surge       = 1
    max_unavailable = 0
  }
}

# Cloud SQL (Primary with read replicas)
resource "random_password" "db_password" {
  length  = 32
  special = true
}

resource "google_sql_database_instance" "primary" {
  name             = "agentstack-db-primary"
  project          = var.project_id
  database_version = "POSTGRES_15"
  region           = local.primary_region.location

  settings {
    tier              = var.database_config.tier
    disk_size         = var.database_config.disk_size_gb
    disk_autoresize   = var.database_config.disk_autoresize
    availability_type = var.database_config.availability_type
    disk_type         = "PD_SSD"

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = google_compute_network.main.id
      enable_private_path_for_google_cloud_services = true
    }

    backup_configuration {
      enabled                        = var.database_config.backup_enabled
      point_in_time_recovery_enabled = var.database_config.point_in_time_recovery
      start_time                     = "02:00"
      location                       = "eu"
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 30
        retention_unit   = "COUNT"
      }
    }

    maintenance_window {
      day          = 7 # Sunday
      hour         = 3
      update_track = "stable"
    }

    database_flags {
      name  = "log_checkpoints"
      value = "on"
    }
    database_flags {
      name  = "log_connections"
      value = "on"
    }
    database_flags {
      name  = "log_disconnections"
      value = "on"
    }
    database_flags {
      name  = "log_duration"
      value = "on"
    }
    database_flags {
      name  = "log_statement"
      value = "ddl"
    }
    database_flags {
      name  = "pgaudit.log"
      value = "all"
    }

    insights_config {
      query_insights_enabled  = true
      query_plans_per_minute  = 5
      query_string_length     = 4500
      record_application_tags = true
      record_client_address   = true
    }

    user_labels = local.labels
  }

  deletion_protection = true

  depends_on = [google_service_networking_connection.private_vpc]
}

# Read replicas in other regions
resource "google_sql_database_instance" "replicas" {
  for_each = { for r in var.regions : r.name => r if !r.primary }

  name                 = "agentstack-db-replica-${each.value.name}"
  project              = var.project_id
  database_version     = "POSTGRES_15"
  region               = each.value.location
  master_instance_name = google_sql_database_instance.primary.name

  replica_configuration {
    failover_target = false
  }

  settings {
    tier            = var.database_config.tier
    disk_size       = var.database_config.disk_size_gb
    disk_autoresize = var.database_config.disk_autoresize
    disk_type       = "PD_SSD"

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = google_compute_network.main.id
      enable_private_path_for_google_cloud_services = true
    }

    user_labels = local.labels
  }

  deletion_protection = true

  depends_on = [google_sql_database_instance.primary]
}

# Database and user
resource "google_sql_database" "main" {
  name     = "agentstack"
  project  = var.project_id
  instance = google_sql_database_instance.primary.name
}

resource "google_sql_user" "main" {
  name     = "agentstack"
  project  = var.project_id
  instance = google_sql_database_instance.primary.name
  password = random_password.db_password.result
}

# Memorystore Redis per region
resource "google_redis_instance" "regional" {
  for_each = { for r in var.regions : r.name => r }

  name           = "agentstack-redis-${each.value.name}"
  project        = var.project_id
  tier           = var.redis_config.tier
  memory_size_gb = var.redis_config.memory_size_gb
  region         = each.value.location
  replica_count  = var.redis_config.replica_count

  authorized_network = google_compute_network.main.id
  connect_mode       = "PRIVATE_SERVICE_ACCESS"

  redis_version = "REDIS_7_0"

  auth_enabled            = true
  transit_encryption_mode = "SERVER_AUTHENTICATION"

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 3
        minutes = 0
      }
    }
  }

  labels = local.labels

  depends_on = [google_service_networking_connection.private_vpc]
}

# Secret Manager for sensitive data
resource "google_secret_manager_secret" "db_password" {
  secret_id = "agentstack-db-password"
  project   = var.project_id

  replication {
    user_managed {
      dynamic "replicas" {
        for_each = var.regions
        content {
          location = replicas.value.location
        }
      }
    }
  }

  labels = local.labels
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = random_password.db_password.result
}

# Global load balancer
resource "google_compute_global_address" "default" {
  name    = "agentstack-global-ip"
  project = var.project_id
}

# Outputs
output "vpc_network" {
  value = google_compute_network.main.name
}

output "gke_clusters" {
  value = { for k, v in google_container_cluster.regional : k => {
    name     = v.name
    location = v.location
    endpoint = v.endpoint
  }}
}

output "database_connection" {
  value = {
    primary  = google_sql_database_instance.primary.connection_name
    replicas = { for k, v in google_sql_database_instance.replicas : k => v.connection_name }
  }
  sensitive = true
}

output "redis_instances" {
  value = { for k, v in google_redis_instance.regional : k => {
    host = v.host
    port = v.port
  }}
  sensitive = true
}

output "global_ip" {
  value = google_compute_global_address.default.address
}
