package service

import (
	"regexp"
	"strings"
)

var (
	// Regex to remove displacement suffixes (e.g., 150cc, 150ccs, 150 cc)
	displacementRegex = regexp.MustCompile(`(?i)\b(\d+)\s*ccs?\b`)
	// Regex to strip punctuation marks (e.g. F.I. -> FI, Honda! -> Honda)
	punctuationRegex = regexp.MustCompile(`[.,!?'":;/()\[\]{}*&^%$#@~<>_+=|\\]`)
	// Regex to replace any remaining non-alphanumeric characters (except hyphens and spaces) with a space
	nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9\-\s]`)
	// Regex to collapse consecutive spaces or hyphens into a single hyphen
	multiHyphenSpaceRegex = regexp.MustCompile(`[\s\-]+`)
)

// Brand prefix mappings for common Indonesian motorcycle models
var brandAliasMap = map[string]string{
	// Honda
	"vario":   "honda-",
	"beat":    "honda-",
	"scoopy":  "honda-",
	"pcx":     "honda-",
	"adv":     "honda-",
	"cbr":     "honda-",
	"cb150":   "honda-",
	"crf":     "honda-",
	"supra":   "honda-",
	"revo":    "honda-",
	"genio":   "honda-",
	"stylo":   "honda-",

	// Yamaha
	"nmax":    "yamaha-",
	"aerox":   "yamaha-",
	"lexi":    "yamaha-",
	"fazzio":  "yamaha-",
	"filano":  "yamaha-",
	"r15":     "yamaha-",
	"r25":     "yamaha-",
	"mt15":    "yamaha-",
	"mt25":    "yamaha-",
	"vixion":  "yamaha-",
	"mx-king": "yamaha-",
	"mio":     "yamaha-",
	"xsr":     "yamaha-",

	// Suzuki
	"nex":     "suzuki-",
	"address": "suzuki-",
	"burgman": "suzuki-",
	"satria":  "suzuki-",
	"gsx":     "suzuki-",

	// Kawasaki
	"ninja":   "kawasaki-",
	"klx":     "kawasaki-",
	"w175":    "kawasaki-",
	"zx25r":   "kawasaki-",

	// Vespa
	"vespa":     "vespa-",
	"sprint":    "vespa-",
	"primavera": "vespa-",
	"lx125":     "vespa-",
}

var knownBrands = []string{"honda", "yamaha", "suzuki", "kawasaki", "vespa"}

// NormalizeQuery transforms raw user input into a deterministic canonical slug.
func NormalizeQuery(query string) string {
	// 1. Lowercase and trim
	cleaned := strings.ToLower(strings.TrimSpace(query))

	// 2. Strip standalone displacement suffixes (e.g., 150cc -> 150)
	cleaned = displacementRegex.ReplaceAllString(cleaned, "$1")

	// 3. Strip punctuation marks (e.g., F.I. -> FI, don't split acronyms with dots)
	cleaned = punctuationRegex.ReplaceAllString(cleaned, "")

	// 4. Remove any remaining non-alphanumeric characters (keep space and hyphen)
	cleaned = nonAlphaNumRegex.ReplaceAllString(cleaned, " ")

	// 4. Collapse consecutive spaces/hyphens to a single hyphen
	slug := multiHyphenSpaceRegex.ReplaceAllString(cleaned, "-")
	slug = strings.Trim(slug, "-")

	if slug == "" {
		return ""
	}

	// 5. Check if the slug already starts with any known brand
	hasBrand := false
	for _, brand := range knownBrands {
		if strings.HasPrefix(slug, brand+"-") || slug == brand {
			hasBrand = true
			break
		}
	}

	// If brand is omitted, check brand alias mapping against tokens
	if !hasBrand {
		for alias, prefix := range brandAliasMap {
			if strings.HasPrefix(slug, alias+"-") || slug == alias || strings.Contains(slug, "-"+alias+"-") || strings.HasSuffix(slug, "-"+alias) {
				slug = prefix + slug
				break
			}
		}
	}

	return slug
}
