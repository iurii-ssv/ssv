package networkconfig

import (
	"math/big"

	spectypes "github.com/bloxapp/ssv-spec/types"

	"github.com/bloxapp/ssv/protocol/v2/blockchain/beacon"
)

var Mainnet = NetworkConfig{
	Name:                 "mainnet",
	Beacon:               beacon.NewNetwork(spectypes.MainNetwork),
	Domain:               spectypes.GenesisMainnet,
	GenesisEpoch:         218450,
	RegistrySyncOffset:   new(big.Int).SetInt64(17507487),
	RegistryContractAddr: "0xDD9BC35aE942eF0cFa76930954a156B3fF30a4E1",
	Bootnodes: []string{
		// Blox
		"enr:-Li4QHEPYASj5ZY3BXXKXAoWcoIw0ChgUlTtfOSxgNlYxlmpEWUR_K6Nr04VXsMpWSQxWWM4QHDyypnl92DQNpWkMS-GAYiWUvo8h2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhCzmKVSJc2VjcDI1NmsxoQOW29na1pUAQw4jF3g0zsPgJG89ViHJOOkHFFklnC2UyIN0Y3CCE4qDdWRwgg-i",

		// 0NEinfra bootnode
		"enr:-Li4QDwrOuhEq5gBJBzFUPkezoYiy56SXZUwkSD7bxYo8RAhPnHyS0de0nOQrzl-cL47RY9Jg8k6Y_MgaUd9a5baYXeGAYnfZE76h2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhDaTS0mJc2VjcDI1NmsxoQMZzUHaN3eClRgF9NAqRNc-ilGpJDDJxdenfo4j-zWKKYN0Y3CCE4iDdWRwgg-g",

		// Eridian (eridianalpha.com)
		"enr:-Li4QIzHQ2H82twhvsu8EePZ6CA1gl0_B0WWsKaT07245TkHUqXay-MXEgObJB7BxMFl8TylFxfnKNxQyGTXh-2nAlOGAYuraxUEh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBKCzUSJc2VjcDI1NmsxoQNKskkQ6-mBdBWr_ORJfyHai5uD0vL6Fuw90X0sPwmRsoN0Y3CCE4iDdWRwgg-g",

		// CryptoManufaktur
		"enr:-Li4QH7FwJcL8gJj0zHAITXqghMkG-A5bfWh2-3Q7vosy9D1BS8HZk-1ITuhK_rfzG3v_UtBDI6uNJZWpdcWfrQFCxKGAYnQ1DRCh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLb3g2Jc2VjcDI1NmsxoQKeSDcZWSaY9FC723E9yYX1Li18bswhLNlxBZdLfgOKp4N0Y3CCE4mDdWRwgg-h",
	},
	WhitelistedOperatorKeys: []string{
		// Blox
		"LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBeSttZUZmOGwyZ2lzK01YU1pXc3oKTFhPTjRodXJ1ak4zQnNSRTJCU1FuV3RMUHY1Y2prSW0xa2FxRThqWEtBbU5nRkwrT3o0eldpczhFRVFhMjB1UgpyM01RM1NMQnlpaWFRYjNDeStjMFg1UDFsTFBBYzVxNnhiZlJBZXp3K2dUUkFYSXo4RXBwaGdVblNyVUQvOXp2CmZ1OFRaQkVLSlYrcnFDRTZZN0FpcU9jVUsrNHF3TWUyeWQrMW9rRld2d2E3c3h4T2VZNGdBcG9jTENNQmRzKzIKQlY2UVR5aVZaT1daQlhFSjdXMllINHBHMWRlMHdMRUZaUnVkcmE1L3RXUzBqSzRRV0Vhc21WeG5LOUpsSWJDdgplZW5vYWt2M1pjamM4WGs1MmRLWGFuNy9TNDNxdHRJT1MvbFdmRDdxSTZvWXp5aXJhZVh2dDdYbXlhODJIa3JZCkd3SURBUUFCCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0K",
		"LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBeTd0dlFKa01nZUx3ekp5c1lZUlgKRWNyTC9PUHBScDkyUWxEdHZNaXQrdXZKbEtSV1B6OFZCbERtV1UvV0UrTkUxa2FyTDVrRFAxazRMMUFUUjJFWgpLNEpMUUp3RW5kNHVBWlB3Rm1UVTEvWmVDN1kvVmlNcmlyMW1pSzRmcXNnTko5UmVWWjAzQ3hpVGNQQjNHNTE0ClhQaklzaUo0eS8wSlB6cmhQckR5Vmt3SnEwWWRnMWpJMUJkbzVaVm15SkZ4eC9lblcwcVUrNG9iaElGZThlUkEKdjUrbS9aa1lUbnNoMklsVk10UjB5TUQwR0I0YWo3MGQ1VVIwMk1yZkhCWXVLOHpnSitXVkN2R0JVTm9ramVFZQpvWVRsYmQwSzAxRWh1MHN1cStjc0FubU8vaTBaaDVHOVM3MU5EVkc4QnBhdVk5cHYvcFlDa3ZqaHNRdGtQTEJKCjd3SURBUUFCCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0K",

		// ssvscan.io (DragonStake)
		"LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBdlMzclA0QWRoSklqK0J1Qm84QmsKK0RTemZ1TDFaNit0b2ZyYWVHbW90N1ltUVFUZHhaRGEwWE9haVhJNU9mWERJbEQ5VUp0SC9yZWlWSFhTNTJuRwp5K2NvY3NuNnZNdnRtdUlvS1JYRGxnYnI4UStod0lpR004K1RrWGFxOVpCRVVSNHpLUnZ1YlpSanZBemhjeDZ3CnU4TzJER1F5MyswTVY1WmtYL0FEYkhpVlJRcWpXWGZWUm1oNmV3S3hhVDNqNC9lei9EMnNPNUtLTXBWUFRXUHMKT2ZFZEdjZjJFSnpoU1Zha0hZOFpuZ0JUMEhIK0ZMeVVVT1prcTRBa25UWVByS1ZyVlBMVmlUaG1Va0hSTUs5VwpVRnNPbFliZlpyeTRHTTdWcEtnSzZGcDd2K1NmZEZuYWUvU0d6d3MzSndIMjFRK0NjV1hRbEFZOGNPbFI0dko4CkpRSURBUUFCCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0K",

		// HellmanAlert
		"LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBcU5Sd0xWSHNWMEloUjdjdUJkb1AKZnVwNTkydEJFSG0vRllDREZQbERMR2NVZ2NzZ29PdHBsV2hMRjBGSzIwQ3ppVi83WVZzcWpxcDh3VDExM3pBbQoxOTZZRlN6WmUzTFhOQXFRWlBwbDlpOVJxdVJJMGlBT2xiWUp0ampJRjd2ZVZLbVdybzMwWTZDV3JPcHpVQ1BPClRGVEpGZ0hvZmtQT2pabmprNURtdDg2ZURveUxzenJQZWQ0LzlyR2NNVUp4WnJBSjEvbFR1ajNaWWVJUk0wS04KUVQ0eitPb3p0T0dBeDVVcUk2THpQL3NGOWRJM3BzM3BIb3dXOWF2RHp3Qm94Y3hWam14NWhRMXowOTN4MnlkYgpWcjgxNDgzTzdqUkt6eFpXeEduOFJzZUROZkxwSi93VFJiQ0lVOFhwUC9IKzd6TWNGMG1HbVlUcjAvcWR1bVNsCjNRSURBUUFCCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0K",
		"LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBdmRWVVJ0OFgxbFA5VDVSUUdYdVkKcFpZWjVBb3VuSEdUakMvQ1FoTmQ5RC9kT2kvSDUwVW1PdVBpTzhYYUF4UFRGcGIrZ2xCeGJRRHVQUGN1cENPdQpKN09lVTBvdzdsQjVMclZlWWt3RExnSHY3bDQwcjRWVTM3NlFueGhuS0JyVHNkaWdmZHJYUWZveGRhajVQQ0VYCnFjK1ozNXFPUmpCZ3dublRlbEJjc2NLMHorSkJaQzU0OXFOWThMbm9aMTBuRFptdW1YVDlac3dISCtJVkZacDYKMEZTY0k0V1V5U1gxVnJJT2tSandoSWlCSFk3YkhrZ01Bci9xeStuRmlFUUVRV2Q2VXAwOWtkS0hNVmdtVFp4KwprQXZRbFZ0Z3luYkFPWkNMeng0Ymo1Yi9MQklIejNiTk9zWlNtR3AxWi9hWDFkd1BaMlhOai83elovNGpuM095CkdRSURBUUFCCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0K",
	},
	PermissionlessActivationEpoch: 249056, // Dec-13-2023 09:58:47 AM UTC
}
