resource "utils_consul_key" "example" {
  path   = "example/key"
  value  = "example-value"
  delete = true
}
