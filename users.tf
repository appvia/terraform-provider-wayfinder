resource "wayfinder_user" "admin" {
  metadata {
    name = "admin-at-appvia.io"
  }

  spec {
    disabled = false
    username = "admin@appvia.io"
  }
}

resource "wayfinder_user" "rohith" {
  metadata {
    name = "rohith.jayawardene-at-appvia.io"
  }

  spec {
    disabled = false
    username = "rohith.jayawardene@appvia.io"
  }
}

resource "wayfinder_user" "kashif" {
  metadata {
    name = "kashif.saadat-at-appvia.io"
  }

  spec {
    disabled = false
    username = "kashif.saadat@appvia.io"
  }
}

resource "wayfinder_user" "marcin" {
  metadata {
    name = "marcin.ciszak-at-appvia.io"
  }

  spec {
    disabled = false
    username = "marcin.ciszak@appvia.io"
  }
}
