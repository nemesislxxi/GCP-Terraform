package google

import (
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDnsRecordSet_importBasic(t *testing.T) {
	t.Parallel()

	zoneName := fmt.Sprintf("dnszone-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsManagedZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDnsRecordSet_basic(zoneName, "127.0.0.10", 300),
			},

			resource.TestStep{
				ResourceName:      "google_dns_record_set.foobar",
				ImportStateId:     zoneName + "/qa.test.com./A",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
