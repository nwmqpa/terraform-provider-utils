resource "utils_consul_exported_service" "example" {
  peer_name         = "other-cluster"
  service_to_export = "logging-service"
}
