resource "wayfinder_manifest" "developers" {
  manifest = <<-YAML
    apiVersion: org/v3
    kind: Group
    metadata:
      name: developers
    spec:
      description: Development team members
      members:
        - kind: User
          name: alice@example.com
  YAML
}
