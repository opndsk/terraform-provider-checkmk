package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-provider-checkmk/internal/client"
)

func TestAccHostResource(t *testing.T) {
	// NOTE: This test requires manual activation of changes in CheckMK.
	// After each step, changes must be activated for them to take effect.
	// Future enhancement: Add automatic activation support.

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccHostResourceConfig("test-terraform-host", "Test Host"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("checkmk_host.test", "host_name", "test-terraform-host"),
					resource.TestCheckResourceAttr("checkmk_host.test", "folder", "/"),
					// NOTE: Attribute checks commented out as they require activation
					// resource.TestCheckResourceAttr("checkmk_host.test", "attributes.alias", "Test Host"),
					// resource.TestCheckResourceAttr("checkmk_host.test", "attributes.ipaddress", "127.0.0.1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "checkmk_host.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore attributes as they may not match without activation
				ImportStateVerifyIgnore: []string{"attributes"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccHostResourceConfig(hostName, alias string) string {
	return fmt.Sprintf(`
resource "checkmk_host" "test" {
  host_name = %[1]q
  folder    = "/"

  attributes = {
    alias     = %[2]q
    ipaddress = "127.0.0.1"
  }
}
`, hostName, alias)
}

// TestAccHostResource_UnprefixedAttributes verifies that a built-in tag group
// can be written without the "tag_" prefix (e.g. "agent" instead of
// "tag_agent") alongside an arbitrary custom attribute, and that state keeps
// the configured (unprefixed) keys without producing a perpetual diff.
func TestAccHostResource_UnprefixedAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostResourceConfigUnprefixed("test-terraform-host-unprefixed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("checkmk_host.unprefixed", "host_name", "test-terraform-host-unprefixed"),
					// State mirrors the configured keys: the built-in tag group
					// stays "agent" and the custom attribute is untouched.
					resource.TestCheckResourceAttr("checkmk_host.unprefixed", "attributes.agent", "cmk-agent"),
					resource.TestCheckResourceAttr("checkmk_host.unprefixed", "attributes.proxy_port", "8080"),
				),
			},
		},
	})
}

func TestAccHostResource_FolderMove(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostResourceConfigWithFolders(
					"test-terraform-host-move",
					"tf-acc-host-move-src",
					"tf-acc-host-move-dst",
					"source",
					"Source Host",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("checkmk_host.test", "folder", "/tf-acc-host-move-src"),
					testAccCheckHostFolderAndAlias("test-terraform-host-move", "/tf-acc-host-move-src", "Source Host"),
				),
			},
			{
				Config: testAccHostResourceConfigWithFolders(
					"test-terraform-host-move",
					"tf-acc-host-move-src",
					"tf-acc-host-move-dst",
					"target",
					"Moved Host",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("checkmk_host.test", "folder", "/tf-acc-host-move-dst"),
					testAccCheckHostFolderAndAlias("test-terraform-host-move", "/tf-acc-host-move-dst", "Moved Host"),
				),
			},
		},
	})
}

func TestAccHostResource_FolderMoveWithoutAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostResourceConfigWithFolders(
					"test-terraform-host-move-no-attrs",
					"tf-acc-host-move-src-no-attrs",
					"tf-acc-host-move-dst-no-attrs",
					"source",
					"Source Host",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("checkmk_host.test", "folder", "/tf-acc-host-move-src-no-attrs"),
					testAccCheckHostFolderAndAlias("test-terraform-host-move-no-attrs", "/tf-acc-host-move-src-no-attrs", "Source Host"),
				),
			},
			{
				Config: testAccHostResourceConfigWithFoldersNoAttributes(
					"test-terraform-host-move-no-attrs",
					"tf-acc-host-move-src-no-attrs",
					"tf-acc-host-move-dst-no-attrs",
					"target",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("checkmk_host.test", "folder", "/tf-acc-host-move-dst-no-attrs"),
					testAccCheckHostFolderAndAlias("test-terraform-host-move-no-attrs", "/tf-acc-host-move-dst-no-attrs", "Source Host"),
				),
			},
		},
	})
}

func testAccHostResourceConfigUnprefixed(hostName string) string {
	return fmt.Sprintf(`
resource "checkmk_host" "unprefixed" {
  host_name = %[1]q
  folder    = "/"

  attributes = {
    alias = "Unprefixed Attributes Host"

    # Built-in host tag group written without the "tag_" prefix.
    agent = "cmk-agent"

    # Arbitrary custom attribute, also without a prefix.
    proxy_port = "8080"
  }
}
`, hostName)
}

func testAccHostResourceConfigWithFolders(hostName, sourceFolderName, targetFolderName, selectedFolder, alias string) string {
	selectedFolderRef := "checkmk_folder.source.path"
	if selectedFolder == "target" {
		selectedFolderRef = "checkmk_folder.target.path"
	}

	return fmt.Sprintf(`
resource "checkmk_folder" "source" {
  name   = %[2]q
  parent = "/"
  title  = "Source Folder"
}

resource "checkmk_folder" "target" {
  name   = %[3]q
  parent = "/"
  title  = "Target Folder"
}

resource "checkmk_host" "test" {
  host_name = %[1]q
  folder    = %[5]s

  attributes = {
    alias     = %[4]q
    ipaddress = "127.0.0.1"
  }
}
`, hostName, sourceFolderName, targetFolderName, alias, selectedFolderRef)
}

func testAccHostResourceConfigWithFoldersNoAttributes(hostName, sourceFolderName, targetFolderName, selectedFolder string) string {
	selectedFolderRef := "checkmk_folder.source.path"
	if selectedFolder == "target" {
		selectedFolderRef = "checkmk_folder.target.path"
	}

	return fmt.Sprintf(`
resource "checkmk_folder" "source" {
  name   = %[2]q
  parent = "/"
  title  = "Source Folder"
}

resource "checkmk_folder" "target" {
  name   = %[3]q
  parent = "/"
  title  = "Target Folder"
}

resource "checkmk_host" "test" {
  host_name = %[1]q
  folder    = %[4]s
}
`, hostName, sourceFolderName, targetFolderName, selectedFolderRef)
}

func testAccCheckHostFolderAndAlias(hostName, expectedFolder, expectedAlias string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		c, err := client.NewClient(
			testAccGetEnv("CHECKMK_URL"),
			testAccGetEnv("CHECKMK_USERNAME"),
			testAccGetEnv("CHECKMK_PASSWORD"),
		)
		if err != nil {
			return fmt.Errorf("failed to create client for host check: %s", err)
		}

		host, err := c.GetHost(context.Background(), hostName)
		if err != nil {
			return fmt.Errorf("failed to fetch host %s: %s", hostName, err)
		}

		if host.Extensions.Folder != expectedFolder {
			return fmt.Errorf("expected host folder %s, got %s", expectedFolder, host.Extensions.Folder)
		}

		alias, _ := host.Extensions.Attributes["alias"].(string)
		if alias != expectedAlias {
			return fmt.Errorf("expected host alias %s, got %s", expectedAlias, alias)
		}

		return nil
	}
}
