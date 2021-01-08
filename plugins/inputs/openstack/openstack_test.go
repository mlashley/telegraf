package openstack_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/influxdata/telegraf/plugins/inputs/openstack"
	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// We take some liberties here with mocking the various Openstack services which are typically exposed on multiple different ports
		// and fake them all from a single mock-server.
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`
			{"versions": {"values": [{"id": "v3.14", "status": "stable", "updated": "2020-04-07T00:00:00Z", "links": [{"rel": "self", "href": "http://` + r.Host + `/v3/"}], "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}]}]}}
			`))
		} else if r.URL.Path == "/v3/auth/tokens" {
			w.WriteHeader(201)
			_, _ = w.Write([]byte(strings.ReplaceAll(keystone_auth_template, "_URL:PORT_", r.Host)))
		} else if r.URL.Path == "/v3/services" {
			_, _ = w.Write([]byte(strings.ReplaceAll(keystone_service_template, "_URL:PORT_", r.Host)))
		} else if r.URL.Path == "/v3/projects" {
			_, _ = w.Write([]byte(strings.ReplaceAll(keystone_project_template, "_URL:PORT_", r.Host)))
		} else if r.URL.Path == "/v2.1/0a6578bd69454ba1a497daa853a77483/os-hypervisors/detail" {
			_, _ = w.Write([]byte(strings.ReplaceAll(nova_hypervisors_template, "_URL:PORT_", r.Host)))
		} else if r.URL.Path == "/v2.1/0a6578bd69454ba1a497daa853a77483/flavors/detail" {
			_, _ = w.Write([]byte(strings.ReplaceAll(nova_flavors_template, "_URL:PORT_", r.Host)))
		} else if r.URL.Path == "/v2.1/0a6578bd69454ba1a497daa853a77483/servers/detail" { // ignoring param ?all_tenants=true
			_, _ = w.Write([]byte(strings.ReplaceAll(nova_servers_template, "_URL:PORT_", r.Host)))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fakeServer.Close()

	plugin := &openstack.OpenStack{
		IdentityEndpoint: fakeServer.URL,
		Username:         "user",
		Password:         "password",
		Domain:           "default",
	}

	var acc testutil.Accumulator
	require.NoError(t, acc.GatherError(plugin.Gather))

	require.Len(t, acc.Metrics, 4)
	fields := map[string]interface{}{
		"projects": 3,
	}

	acc.AssertContainsTaggedFields(t, "openstack_identity", fields, map[string]string{})

	fields = map[string]interface{}{
		"memory_mb":      15872,
		"memory_mb_used": 1024,
		"running_vms":    1,
		"vcpus":          8,
		"vcpus_used":     1,
	}
	tags := map[string]string{
		"name": "hypervisor.hostname.com",
	}
	acc.AssertContainsTaggedFields(t, "openstack_hypervisor", fields, tags)

	fields = map[string]interface{}{
		"status":  "error",
		"vcpus":   1,
		"ram_mb":  512,
		"disk_gb": 1,
	}
	tags = map[string]string{
		"name":    "testvm-from-volume",
		"project": "admin",
	}
	acc.AssertContainsTaggedFields(t, "openstack_server", fields, tags)

	fields = map[string]interface{}{
		"status":  "shutoff",
		"vcpus":   1,
		"ram_mb":  512,
		"disk_gb": 1,
	}
	tags = map[string]string{
		"name":    "test2",
		"project": "admin",
	}
	acc.AssertContainsTaggedFields(t, "openstack_server", fields, tags)

}

const keystone_auth_template = `
{
	"token": {
		"methods": [
			"password"
		],
		"user": {
			"domain": {
				"id": "default",
				"name": "Default"
			},
			"id": "1888fa2cbe2d47359652fffbafc013e0",
			"name": "admin",
			"password_expires_at": null
		},
		"audit_ids": [
			"9-EJMb1tT1u1nW0Xo9yy3w"
		],
		"expires_at": "2021-01-07T22:34:40.000000Z",
		"issued_at": "2021-01-07T21:34:40.000000Z",
		"project": {
			"domain": {
				"id": "default",
				"name": "Default"
			},
			"id": "0a6578bd69454ba1a497daa853a77483",
			"name": "admin"
		},
		"is_domain": false,
		"roles": [
			{
				"id": "f222fcc20ee646438866bf8e5527cce3",
				"name": "member"
			},
			{
				"id": "7307c67ce33240a8aace2c28cfc8b01d",
				"name": "reader"
			},
			{
				"id": "94c9921b6f424a3dbc86f0d067485255",
				"name": "admin"
			}
		],
		"catalog": [
			{
				"endpoints": [
					{
						"id": "04a594d3d1d54f2e9734f2b7b7e44b61",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "e063c6a37c6c48cf9ff5f15b8cde6edb",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "f0348f3809b742e9993b511b2a387294",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					}
				],
				"id": "06d9887ad8af4e15877d3a1c73fd56dd",
				"type": "alarming",
				"name": "aodh"
			},
			{
				"endpoints": [
					{
						"id": "3f6ebd5528454799aebf08c56abc1bd5",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v3/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "64058025f0a44504b257df166b734964",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v3/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "db78f5c56448440da68db3e27bc69515",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v3/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					}
				],
				"id": "26b93ded7e904aeebff9382dd3ae6b0f",
				"type": "volumev3",
				"name": "cinderv3"
			},
			{
				"endpoints": [
					{
						"id": "3c78ab7ad92b4cf7ae3931a93e7b65b2",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v2/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "76925b251f674cae842a31e251a1ab4a",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v2/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "dfb01e7191ca4aadbaef8cf67574c8e4",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v2/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					}
				],
				"id": "297d589a75334b69be34946bea56c663",
				"type": "volumev2",
				"name": "cinderv2"
			},
			{
				"endpoints": [
					{
						"id": "2d0664d85199432f9242e405d8dc3302",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "813f9f3a59f941cd846ba846cb90597a",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "c6e9ecb1b4c047798b7803d2b9990be0",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					}
				],
				"id": "42d9e24c64be48be8ec23d453e2f82f9",
				"type": "metric",
				"name": "gnocchi"
			},
			{
				"endpoints": [
					{
						"id": "1a0b6c96fe3d443785ef1857d5ecea0b",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "67916c6e42f94e38b48806eade91c90c",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "ef45820ab55e4cdc9988862d9ab650cf",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					}
				],
				"id": "719e9b17302b4397ad222dc84a0968e7",
				"type": "compute",
				"name": "nova"
			},
			{
				"endpoints": [
					{
						"id": "35f1526dcf7b47109003f9337dc9a3fb",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "641bb6a1029741019077bc6788b64908",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "f5a5b28d3b72405481c7e803c39430c8",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					}
				],
				"id": "87051d4f393e46c3b97ff0cae913de0e",
				"type": "identity",
				"name": "keystone"
			},
			{
				"endpoints": [
					{
						"id": "8a0741b0b8f74774b21e0acf8ff27d6f",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "970e43343efa498e96931c694ff3652f",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "c250123cdb304e128c310e8f510b8d4d",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					}
				],
				"id": "aabadadfcceb40529d0c0b9a7bf63c8c",
				"type": "image",
				"name": "glance"
			},
			{
				"endpoints": [
					{
						"id": "0d238d5a8bc34719becdd5402418ba86",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "1a3be818d829434fa0e7d39eef86a228",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "68e9306f7f8f4d38a2a4bc15d897c5f0",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					}
				],
				"id": "ad5ed8cafe544e3e9510a41081e0a40d",
				"type": "network",
				"name": "neutron"
			},
			{
				"endpoints": [
					{
						"id": "3b8184ce06fb4e78ad6176407bf7310e",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "9885ddb810c74f2980d9adfa1e2d8d1e",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					},
					{
						"id": "dd34a72c1f7f441e885a5db142d1c061",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_",
						"region": "RegionOne"
					}
				],
				"id": "b50270f230e7434e93813a69c78dbb2f",
				"type": "metering",
				"name": "ceilometer"
			},
			{
				"endpoints": [
					{
						"id": "dba2d9212fcb44e1b73e102f4a33cb13",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v1/AUTH_0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "e8f7d4b5f1744da8afff4ab433bcbec3",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v1/AUTH_0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					},
					{
						"id": "f8666a94337f4a64aea611f9f780029d",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/v1/AUTH_0a6578bd69454ba1a497daa853a77483",
						"region": "RegionOne"
					}
				],
				"id": "c1ee532fed1346c3b49193f7fe8b387b",
				"type": "object-store",
				"name": "swift"
			},
			{
				"endpoints": [
					{
						"id": "48e61d862a5f41b2a5d3d90e89e77b6f",
						"interface": "public",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/placement",
						"region": "RegionOne"
					},
					{
						"id": "ba637c0f0af1403f97419c0a2e2cd146",
						"interface": "admin",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/placement",
						"region": "RegionOne"
					},
					{
						"id": "ebfac4f679544051a3a2d1742e5a06f0",
						"interface": "internal",
						"region_id": "RegionOne",
						"url": "http://_URL:PORT_/placement",
						"region": "RegionOne"
					}
				],
				"id": "f273ccb387bc4de9b36434e10cb99c92",
				"type": "placement",
				"name": "placement"
			}
		]
	}
}
`

const keystone_service_template = `
{
	"services": [
		{
			"name": "aodh",
			"description": "OpenStack Alarming Service",
			"id": "06d9887ad8af4e15877d3a1c73fd56dd",
			"type": "alarming",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/06d9887ad8af4e15877d3a1c73fd56dd"
			}
		},
		{
			"name": "cinderv3",
			"description": "Cinder Service v3",
			"id": "26b93ded7e904aeebff9382dd3ae6b0f",
			"type": "volumev3",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/26b93ded7e904aeebff9382dd3ae6b0f"
			}
		},
		{
			"name": "cinderv2",
			"description": "Cinder Service v2",
			"id": "297d589a75334b69be34946bea56c663",
			"type": "volumev2",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/297d589a75334b69be34946bea56c663"
			}
		},
		{
			"name": "gnocchi",
			"description": "OpenStack Metric Service",
			"id": "42d9e24c64be48be8ec23d453e2f82f9",
			"type": "metric",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/42d9e24c64be48be8ec23d453e2f82f9"
			}
		},
		{
			"name": "nova",
			"description": "Openstack Compute Service",
			"id": "719e9b17302b4397ad222dc84a0968e7",
			"type": "compute",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/719e9b17302b4397ad222dc84a0968e7"
			}
		},
		{
			"name": "keystone",
			"id": "87051d4f393e46c3b97ff0cae913de0e",
			"type": "identity",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/87051d4f393e46c3b97ff0cae913de0e"
			}
		},
		{
			"name": "glance",
			"description": "OpenStack Image Service",
			"id": "aabadadfcceb40529d0c0b9a7bf63c8c",
			"type": "image",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/aabadadfcceb40529d0c0b9a7bf63c8c"
			}
		},
		{
			"name": "neutron",
			"description": "Neutron Networking Service",
			"id": "ad5ed8cafe544e3e9510a41081e0a40d",
			"type": "network",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/ad5ed8cafe544e3e9510a41081e0a40d"
			}
		},
		{
			"name": "ceilometer",
			"description": "Openstack Metering Service",
			"id": "b50270f230e7434e93813a69c78dbb2f",
			"type": "metering",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/b50270f230e7434e93813a69c78dbb2f"
			}
		},
		{
			"name": "swift",
			"description": "Openstack Object-Store Service",
			"id": "c1ee532fed1346c3b49193f7fe8b387b",
			"type": "object-store",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/c1ee532fed1346c3b49193f7fe8b387b"
			}
		},
		{
			"name": "placement",
			"description": "Placement Service",
			"id": "f273ccb387bc4de9b36434e10cb99c92",
			"type": "placement",
			"enabled": true,
			"links": {
				"self": "http://_URL:PORT_/v3/services/f273ccb387bc4de9b36434e10cb99c92"
			}
		}
	],
	"links": {
		"next": null,
		"self": "http://_URL:PORT_/v3/services",
		"previous": null
	}
}`

const keystone_project_template = `
{
	"projects": [
		{
			"id": "0a6578bd69454ba1a497daa853a77483",
			"name": "admin",
			"domain_id": "default",
			"description": "Bootstrap project for initializing the cloud.",
			"enabled": true,
			"parent_id": "default",
			"is_domain": false,
			"tags": [],
			"options": {},
			"links": {
				"self": "http://_URL:PORT_/v3/projects/0a6578bd69454ba1a497daa853a77483"
			}
		},
		{
			"id": "68846ce657fc4b60b570c28b78934d80",
			"name": "demo",
			"domain_id": "default",
			"description": "default tenant",
			"enabled": true,
			"parent_id": "default",
			"is_domain": false,
			"tags": [],
			"options": {},
			"links": {
				"self": "http://_URL:PORT_/v3/projects/68846ce657fc4b60b570c28b78934d80"
			}
		},
		{
			"id": "dd0be9663c074613afe2c1256ce5b89a",
			"name": "services",
			"domain_id": "default",
			"description": "",
			"enabled": true,
			"parent_id": "default",
			"is_domain": false,
			"tags": [],
			"options": {},
			"links": {
				"self": "http://_URL:PORT_/v3/projects/dd0be9663c074613afe2c1256ce5b89a"
			}
		}
	],
	"links": {
		"next": null,
		"self": "http://_URL:PORT_/v3/projects",
		"previous": null
	}
}
`

const nova_hypervisors_template = `
{
	"hypervisors": [
		{
			"id": 1,
			"hypervisor_hostname": "hypervisor.hostname.com",
			"state": "up",
			"status": "enabled",
			"vcpus": 8,
			"memory_mb": 15872,
			"local_gb": 16,
			"vcpus_used": 1,
			"memory_mb_used": 1024,
			"local_gb_used": 1,
			"hypervisor_type": "QEMU",
			"hypervisor_version": 4002000,
			"free_ram_mb": 14848,
			"free_disk_gb": 15,
			"current_workload": 0,
			"running_vms": 1,
			"disk_available_least": 11,
			"host_ip": "192.168.1.1",
			"service": {
				"id": 5,
				"host": "hypervisor.hostname.com",
				"disabled_reason": null
			},
			"cpu_info": "{\"arch\": \"x86_64\", \"model\": \"Opteron_G2\", \"vendor\": \"AMD\", \"topology\": {\"cells\": 1, \"sockets\": 8, \"cores\": 1, \"threads\": 1}, \"features\": [\"sse4a\", \"pse36\", \"fpu\", \"sse2\", \"pclmuldq\", \"apic\", \"nx\", \"sse4.2\", \"popcnt\", \"avx2\", \"cx8\", \"sse4.1\", \"mce\", \"tsc\", \"smap\", \"bmi2\", \"tsc_adjust\", \"pdpe1gb\", \"pni\", \"ssse3\", \"aes\", \"adx\", \"rdtscp\", \"mmxext\", \"mca\", \"svm\", \"rdrand\", \"pat\", \"movbe\", \"mds-no\", \"hypervisor\", \"sse\", \"xsaves\", \"msr\", \"clflushopt\", \"bmi1\", \"cmp_legacy\", \"de\", \"abm\", \"wbnoinvd\", \"cmov\", \"mmx\", \"sha-ni\", \"ssbd\", \"sep\", \"f16c\", \"fxsr\", \"misalignsse\", \"pge\", \"cr8legacy\", \"virt-ssbd\", \"clzero\", \"avx\", \"rdseed\", \"arat\", \"cx16\", \"x2apic\", \"mtrr\", \"ibpb\", \"fxsr_opt\", \"clwb\", \"lahf_lm\", \"amd-ssbd\", \"lm\", \"clflush\", \"pse\", \"xgetbv1\", \"osvw\", \"fma\", \"perfctr_core\", \"pae\", \"3dnowprefetch\", \"syscall\", \"fsgsbase\", \"xsave\", \"vme\", \"tsc-deadline\", \"umip\", \"stibp\", \"xsaveopt\", \"skip-l1dfl-vmentry\", \"smep\", \"arch-capabilities\", \"rdctl-no\", \"xsavec\"]}"
		}
	]
}
`

const nova_flavors_template = `
{
	"flavors": [
		{
			"id": "1",
			"name": "m1.tiny",
			"ram": 512,
			"disk": 1,
			"swap": "",
			"OS-FLV-EXT-DATA:ephemeral": 0,
			"OS-FLV-DISABLED:disabled": false,
			"vcpus": 1,
			"os-flavor-access:is_public": true,
			"rxtx_factor": 1.0,
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/flavors/1"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/1"
				}
			]
		},
		{
			"id": "2",
			"name": "m1.small",
			"ram": 2048,
			"disk": 20,
			"swap": "",
			"OS-FLV-EXT-DATA:ephemeral": 0,
			"OS-FLV-DISABLED:disabled": false,
			"vcpus": 1,
			"os-flavor-access:is_public": true,
			"rxtx_factor": 1.0,
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/flavors/2"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/2"
				}
			]
		},
		{
			"id": "3",
			"name": "m1.medium",
			"ram": 4096,
			"disk": 40,
			"swap": "",
			"OS-FLV-EXT-DATA:ephemeral": 0,
			"OS-FLV-DISABLED:disabled": false,
			"vcpus": 2,
			"os-flavor-access:is_public": true,
			"rxtx_factor": 1.0,
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/flavors/3"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/3"
				}
			]
		},
		{
			"id": "4",
			"name": "m1.large",
			"ram": 8192,
			"disk": 80,
			"swap": "",
			"OS-FLV-EXT-DATA:ephemeral": 0,
			"OS-FLV-DISABLED:disabled": false,
			"vcpus": 4,
			"os-flavor-access:is_public": true,
			"rxtx_factor": 1.0,
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/flavors/4"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/4"
				}
			]
		},
		{
			"id": "5",
			"name": "m1.xlarge",
			"ram": 16384,
			"disk": 160,
			"swap": "",
			"OS-FLV-EXT-DATA:ephemeral": 0,
			"OS-FLV-DISABLED:disabled": false,
			"vcpus": 8,
			"os-flavor-access:is_public": true,
			"rxtx_factor": 1.0,
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/flavors/5"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/5"
				}
			]
		}
	]
}
`

const nova_servers_template = `
{
	"servers": [
		{
			"id": "24152c29-a6ba-462a-ba83-6bb81c5e4a90",
			"name": "testvm-from-volume",
			"status": "ERROR",
			"tenant_id": "0a6578bd69454ba1a497daa853a77483",
			"user_id": "1888fa2cbe2d47359652fffbafc013e0",
			"metadata": {},
			"hostId": "",
			"image": "",
			"flavor": {
				"id": "1",
				"links": [
					{
						"rel": "bookmark",
						"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/1"
					}
				]
			},
			"created": "2020-09-06T15:26:07Z",
			"updated": "2020-09-06T15:26:13Z",
			"addresses": {},
			"accessIPv4": "",
			"accessIPv6": "",
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/servers/24152c29-a6ba-462a-ba83-6bb81c5e4a90"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/servers/24152c29-a6ba-462a-ba83-6bb81c5e4a90"
				}
			],
			"OS-DCF:diskConfig": "AUTO",
			"fault": {
				"code": 500,
				"created": "2020-09-06T15:26:13Z",
				"message": "Exceeded maximum number of retries. Exhausted all hosts available for retrying build failures for instance 24152c29-a6ba-462a-ba83-6bb81c5e4a90.",
				"details": "Traceback (most recent call last):\n  File \"/usr/lib/python3.6/site-packages/nova/conductor/manager.py\", line 666, in build_instances\n    raise exception.MaxRetriesExceeded(reason=msg)\nnova.exception.MaxRetriesExceeded: Exceeded maximum number of retries. Exhausted all hosts available for retrying build failures for instance 24152c29-a6ba-462a-ba83-6bb81c5e4a90.\n"
			},
			"OS-EXT-AZ:availability_zone": "",
			"config_drive": "",
			"key_name": null,
			"OS-SRV-USG:launched_at": null,
			"OS-SRV-USG:terminated_at": null,
			"OS-EXT-SRV-ATTR:host": null,
			"OS-EXT-SRV-ATTR:instance_name": "instance-00000006",
			"OS-EXT-SRV-ATTR:hypervisor_hostname": null,
			"OS-EXT-STS:task_state": null,
			"OS-EXT-STS:vm_state": "error",
			"OS-EXT-STS:power_state": 0,
			"os-extended-volumes:volumes_attached": [
				{
					"id": "d6a7d469-2062-41b3-b31c-7fb2fbe027f2"
				}
			]
		},
		{
			"id": "8e73c438-adc3-4c4e-9456-bb923dd09d0c",
			"name": "test2",
			"status": "SHUTOFF",
			"tenant_id": "0a6578bd69454ba1a497daa853a77483",
			"user_id": "1888fa2cbe2d47359652fffbafc013e0",
			"metadata": {},
			"hostId": "4f9b62eb60a34d9c7dbe235409d80e7a5cbfa68cc6f35970898ce11c",
			"image": {
				"id": "f5393b2c-0c9b-489b-befc-448f1eae4a94",
				"links": [
					{
						"rel": "bookmark",
						"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/images/f5393b2c-0c9b-489b-befc-448f1eae4a94"
					}
				]
			},
			"flavor": {
				"id": "1",
				"links": [
					{
						"rel": "bookmark",
						"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/flavors/1"
					}
				]
			},
			"created": "2020-09-06T15:11:42Z",
			"updated": "2020-09-06T15:36:17Z",
			"addresses": {
				"public": [
					{
						"version": 4,
						"addr": "172.24.4.238",
						"OS-EXT-IPS:type": "fixed",
						"OS-EXT-IPS-MAC:mac_addr": "fa:16:3e:9e:0c:c7"
					}
				]
			},
			"accessIPv4": "",
			"accessIPv6": "",
			"links": [
				{
					"rel": "self",
					"href": "http://_URL:PORT_/v2.1/0a6578bd69454ba1a497daa853a77483/servers/8e73c438-adc3-4c4e-9456-bb923dd09d0c"
				},
				{
					"rel": "bookmark",
					"href": "http://_URL:PORT_/0a6578bd69454ba1a497daa853a77483/servers/8e73c438-adc3-4c4e-9456-bb923dd09d0c"
				}
			],
			"OS-DCF:diskConfig": "AUTO",
			"OS-EXT-AZ:availability_zone": "nova",
			"config_drive": "",
			"key_name": null,
			"OS-SRV-USG:launched_at": "2020-09-06T15:11:46.000000",
			"OS-SRV-USG:terminated_at": null,
			"OS-EXT-SRV-ATTR:host": "hypervisor.hostname.com",
			"OS-EXT-SRV-ATTR:instance_name": "instance-00000003",
			"OS-EXT-SRV-ATTR:hypervisor_hostname": "hypervisor.hostname.com",
			"OS-EXT-STS:task_state": null,
			"OS-EXT-STS:vm_state": "stopped",
			"OS-EXT-STS:power_state": 4,
			"os-extended-volumes:volumes_attached": [],
			"security_groups": [
				{
					"name": "default"
				}
			]
		}
	]
}
`
