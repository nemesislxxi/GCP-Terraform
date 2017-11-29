package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceComputeImage_basic(t *testing.T) {
	t.Parallel()

	family := acctest.RandomWithPrefix("tf-test")
	name := acctest.RandomWithPrefix("tf-test")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeImageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceComputeImage_basic(family, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.google_compute_image.from_name",
						"name", name),
					resource.TestCheckResourceAttr("data.google_compute_image.from_name",
						"family", family),
					resource.TestCheckResourceAttrSet("data.google_compute_image.from_name",
						"self_link"),
				),
			},
		},
	})
}

func testAccDataSourceComputeImage_basic(family, name string) string {
	return fmt.Sprintf(`
resource "google_compute_image" "image" {
  family = "%s"
  name = "%s"

  source_disk = "${google_compute_disk.disk.self_link}"
}

resource "google_compute_disk" "disk" {
  name = "%s-disk"
  zone = "us-central1-b"
}

data "google_compute_image" "from_name" {
  name = "${google_compute_image.image.name}"
}

data "google_compute_image" "from_family" {
  family = "${google_compute_image.image.family}"
}
`, family, name, name)
}
