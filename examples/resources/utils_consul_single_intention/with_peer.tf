resource "utils_consul_single_intention" "example" {
  destination_service = "destination"
  source_service      = "source"
  source_peer         = "other-cluster"
}
