// Package core provides routing rule presets for common use cases
package core

// RoutingPreset represents a predefined routing configuration
type RoutingPreset struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Rules       *RoutingRules `json:"rules"`
}

// Available presets
var (
	// PresetDirectCN - Direct connection for China, proxy for everything else
	PresetDirectCN = &RoutingPreset{
		Name:        "direct_cn",
		Description: "Direct connection for Chinese websites, proxy for international traffic",
		Rules: &RoutingRules{
			GeoSiteDirect: []string{"cn", "private", "apple-cn", "microsoft-cn", "google-cn"},
			GeoIPDirect:   []string{"cn", "private"},
			GeoSiteProxy:  []string{"geolocation-!cn"},
			Final:         "proxy",
		},
	}

	// PresetProxyCN - Proxy for China, direct for everything else (for users in China)
	PresetProxyCN = &RoutingPreset{
		Name:        "proxy_cn",
		Description: "Proxy for Chinese websites, direct for international traffic (for users in China)",
		Rules: &RoutingRules{
			GeoSiteProxy:  []string{"cn", "geolocation-cn"},
			GeoIPProxy:    []string{"cn"},
			GeoSiteDirect: []string{"private", "geolocation-!cn"},
			GeoIPDirect:   []string{"private"},
			Final:         "direct",
		},
	}

	// PresetAdBlock - Block ads and trackers
	PresetAdBlock = &RoutingPreset{
		Name:        "adblock",
		Description: "Block ads, trackers and malware",
		Rules: &RoutingRules{
			GeoSiteBlock: []string{
				"category-ads-all",
				"category-ads",
				"category-tracker",
				"category-malware",
				"category-phishing",
				"category-cryptominer",
			},
			Final: "proxy",
		},
	}

	// PresetStreaming - Optimize for streaming services
	PresetStreaming = &RoutingPreset{
		Name:        "streaming",
		Description: "Route streaming services through proxy, direct for other traffic",
		Rules: &RoutingRules{
			GeoSiteProxy: []string{
				"netflix",
				"disney",
				"hulu",
				"hbo",
				"amazon",
				"youtube",
				"spotify",
				"apple-tv",
				"bbc",
				"itv",
			},
			GeoSiteDirect: []string{"cn", "private"},
			GeoIPDirect:   []string{"cn", "private"},
			Final:         "direct",
		},
	}

	// PresetGaming - Optimize for gaming (low latency)
	PresetGaming = &RoutingPreset{
		Name:        "gaming",
		Description: "Direct connection for gaming platforms, proxy for other traffic",
		Rules: &RoutingRules{
			GeoSiteDirect: []string{
				"steam",
				"epicgames",
				"ea",
				"ubisoft",
				"blizzard",
				"xbox",
				"playstation",
				"nintendo",
				"riot",
				"valorant",
				"leagueoflegends",
			},
			PortDirect: []int{
				27015, 27016, 27017, 27018, 27019, 27020, // Steam
				3074, 3075,                               // Xbox
				9308,                                     // PlayStation
				5000, 5001,                               // Various games
				7777, 7778,                               // Unreal
				25565,                                    // Minecraft
			},
			ProtocolDirect: []string{"udp"},
			Final:          "proxy",
		},
	}

	// PresetPrivacy - Maximum privacy (everything through proxy)
	PresetPrivacy = &RoutingPreset{
		Name:        "privacy",
		Description: "Route all traffic through proxy for maximum privacy",
		Rules: &RoutingRules{
			GeoSiteDirect: []string{"private"},
			GeoIPDirect:   []string{"private"},
			Final:         "proxy",
		},
	}

	// PresetSplitWork - Split tunneling for work
	PresetSplitWork = &RoutingPreset{
		Name:        "split_work",
		Description: "Direct for work-related services, proxy for personal traffic",
		Rules: &RoutingRules{
			GeoSiteDirect: []string{
				"slack",
				"zoom",
				"teams",
				"office365",
				"google-workspace",
				"atlassian",
				"github",
				"gitlab",
				"jira",
				"confluence",
				"trello",
				"asana",
				"notion",
				"figma",
			},
			DomainDirect: []string{
				"*.corp.*",
				"*.company.*",
				"*.internal",
				"*.local",
			},
			Final: "proxy",
		},
	}

	// PresetRussia - Optimized for Russian users
	PresetRussia = &RoutingPreset{
		Name:        "russia",
		Description: "Optimized routing for Russian users (direct for Russian sites, proxy for blocked)",
		Rules: &RoutingRules{
			GeoSiteDirect: []string{
				"ru",
				"yandex",
				"vk",
				"mailru",
				"sberbank",
				"tinkoff",
				"gazprombank",
				"vtb",
				"alfa",
				"raiffeisen",
				"ozon",
				"wildberries",
				"avito",
				"kaspersky",
				"drweb",
			},
			GeoIPDirect: []string{"ru", "private"},
			DomainDirect: []string{
				"*.ru",
				"*.рф",
				"*.xn--p1ai",
			},
			GeoSiteProxy: []string{
				"twitter",
				"facebook",
				"instagram",
				"linkedin",
				"tiktok",
				"discord",
				"telegram",
				"wikipedia",
			},
			Final: "proxy",
		},
	}

	// PresetBelarus - Optimized for Belarusian users
	PresetBelarus = &RoutingPreset{
		Name:        "belarus",
		Description: "Optimized routing for Belarusian users",
		Rules: &RoutingRules{
			GeoSiteDirect: []string{
				"by",
				"tutby",
				"onliner",
			},
			GeoIPDirect: []string{"by", "ru", "private"},
			DomainDirect: []string{
				"*.by",
			},
			Final: "proxy",
		},
	}

	// PresetMinimal - Minimal rules (block only ads)
	PresetMinimal = &RoutingPreset{
		Name:        "minimal",
		Description: "Minimal rules - block ads only, everything else direct",
		Rules: &RoutingRules{
			GeoSiteBlock: []string{
				"category-ads-all",
				"category-ads",
			},
			Final: "direct",
		},
	}

	// PresetSocialMedia - Block social media distractions
	PresetSocialMedia = &RoutingPreset{
		Name:        "no_social",
		Description: "Block social media and entertainment sites",
		Rules: &RoutingRules{
			GeoSiteBlock: []string{
				"twitter",
				"facebook",
				"instagram",
				"tiktok",
				"snapchat",
				"reddit",
				"9gag",
				"twitch",
				"youtube",
				"netflix",
			},
			DomainBlock: []string{
				"*.twitter.com",
				"*.facebook.com",
				"*.instagram.com",
				"*.tiktok.com",
			},
			Final: "direct",
		},
	}

	// PresetTorBlock - Block Tor and VPN detection
	PresetTorBlock = &RoutingPreset{
		Name:        "block_tor",
		Description: "Block Tor exit nodes and VPN detection services",
		Rules: &RoutingRules{
			GeoIPBlock: []string{"tor"},
			DomainBlock: []string{
				"*checkip*",
				"*ipcheck*",
				"*vpnblock*",
				"*proxydetect*",
			},
			Final: "proxy",
		},
	}
)

// GetPreset returns a preset by name
func GetPreset(name string) *RoutingPreset {
	switch name {
	case "direct_cn":
		return PresetDirectCN
	case "proxy_cn":
		return PresetProxyCN
	case "adblock":
		return PresetAdBlock
	case "streaming":
		return PresetStreaming
	case "gaming":
		return PresetGaming
	case "privacy":
		return PresetPrivacy
	case "split_work":
		return PresetSplitWork
	case "russia":
		return PresetRussia
	case "belarus":
		return PresetBelarus
	case "minimal":
		return PresetMinimal
	case "no_social":
		return PresetSocialMedia
	case "block_tor":
		return PresetTorBlock
	default:
		return nil
	}
}

// ListPresets returns all available preset names
func ListPresets() []string {
	return []string{
		"direct_cn",
		"proxy_cn",
		"adblock",
		"streaming",
		"gaming",
		"privacy",
		"split_work",
		"russia",
		"belarus",
		"minimal",
		"no_social",
		"block_tor",
	}
}

// GetPresetWithDescription returns preset name and description pairs
func GetPresetWithDescription() map[string]string {
	return map[string]string{
		"direct_cn":   "Direct CN - Direct for Chinese sites, proxy for international",
		"proxy_cn":    "Proxy CN - Proxy for Chinese sites (for users in China)",
		"adblock":     "AdBlock - Block ads and trackers",
		"streaming":   "Streaming - Optimize for streaming services",
		"gaming":      "Gaming - Low latency for games",
		"privacy":     "Privacy - All traffic through proxy",
		"split_work":  "Split Work - Direct for work services",
		"russia":      "Russia - Optimized for Russian users",
		"belarus":     "Belarus - Optimized for Belarusian users",
		"minimal":     "Minimal - Block ads only",
		"no_social":   "No Social - Block social media",
		"block_tor":   "Block Tor - Block Tor exit nodes",
	}
}

// CreateCustomPreset creates a custom preset from rules
func CreateCustomPreset(name, description string, rules *RoutingRules) *RoutingPreset {
	return &RoutingPreset{
		Name:        name,
		Description: description,
		Rules:       rules,
	}
}

// MergeRules merges multiple routing rules (later rules override earlier ones)
func MergeRules(rules ...*RoutingRules) *RoutingRules {
	result := &RoutingRules{
		Final: "proxy",
	}

	for _, r := range rules {
		if r == nil {
			continue
		}

		result.GeoIPDirect = append(result.GeoIPDirect, r.GeoIPDirect...)
		result.GeoIPProxy = append(result.GeoIPProxy, r.GeoIPProxy...)
		result.GeoIPBlock = append(result.GeoIPBlock, r.GeoIPBlock...)

		result.GeoSiteDirect = append(result.GeoSiteDirect, r.GeoSiteDirect...)
		result.GeoSiteProxy = append(result.GeoSiteProxy, r.GeoSiteProxy...)
		result.GeoSiteBlock = append(result.GeoSiteBlock, r.GeoSiteBlock...)

		result.DomainDirect = append(result.DomainDirect, r.DomainDirect...)
		result.DomainProxy = append(result.DomainProxy, r.DomainProxy...)
		result.DomainBlock = append(result.DomainBlock, r.DomainBlock...)

		result.IPDirect = append(result.IPDirect, r.IPDirect...)
		result.IPProxy = append(result.IPProxy, r.IPProxy...)
		result.IPBlock = append(result.IPBlock, r.IPBlock...)

		result.PortDirect = append(result.PortDirect, r.PortDirect...)
		result.PortProxy = append(result.PortProxy, r.PortProxy...)
		result.PortBlock = append(result.PortBlock, r.PortBlock...)

		result.ProtocolDirect = append(result.ProtocolDirect, r.ProtocolDirect...)
		result.ProtocolProxy = append(result.ProtocolProxy, r.ProtocolProxy...)
		result.ProtocolBlock = append(result.ProtocolBlock, r.ProtocolBlock...)

		if r.Final != "" {
			result.Final = r.Final
		}
	}

	return result
}
