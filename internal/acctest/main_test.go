package acctest

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestMain runs the sweepers with -sweep=<any label>, otherwise the tests.
func TestMain(m *testing.M) {
	AddSweepers()
	resource.TestMain(m)
}
