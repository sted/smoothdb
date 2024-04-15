package database

import (
	"fmt"
	"mime"
	"sort"
	"strings"
)

type mediaType struct {
	Type        string
	Quality     float64
	Specificity int
}

// List of supported content type for output (Accept header)
var supportedOutputContentTypes = []string{
	"application/json",
	"application/vnd.pgrst.object+json",
	"text/csv",
	"application/octet-stream",
}
var defaultOutputContentType = "application/json"

// contentNegotiation to choose the output content type.
// It Supports also q (quality) parameters
func contentNegotiation(acceptHeaders []string) string {
	if len(acceptHeaders) == 0 {
		return defaultOutputContentType
	}

	parsedTypes := parseMediaTypes(acceptHeaders)

	for _, parsedType := range parsedTypes {
		for _, supportedType := range supportedOutputContentTypes {
			if parsedType.Type == supportedType {
				return supportedType
			}
			if matchesWildcard(parsedType.Type, supportedType) {
				return supportedType
			}
		}
	}

	return ""
}

func parseMediaTypes(acceptHeaders []string) []mediaType {
	var types []mediaType

	for _, header := range acceptHeaders {
		for _, part := range strings.Split(header, ",") {
			mt, params, err := mime.ParseMediaType(part)
			if err != nil {
				continue
			}

			q := 1.0
			if qs, ok := params["q"]; ok {
				var qf float64
				_, err := fmt.Sscanf(qs, "%f", &qf)
				if err == nil && qf >= 0 && qf <= 1 {
					q = qf
				}
			}

			specificity := 0
			if mt == "*/*" {
				specificity = 0
			} else if strings.HasSuffix(mt, "/*") {
				specificity = 1
			} else {
				specificity = 2
			}

			types = append(types, mediaType{Type: mt, Quality: q, Specificity: specificity})
		}
	}

	sort.Slice(types, func(i, j int) bool {
		if types[i].Quality == types[j].Quality {
			return types[i].Specificity > types[j].Specificity
		}
		return types[i].Quality > types[j].Quality
	})

	return types
}

func matchesWildcard(parsedType, supportedType string) bool {
	if parsedType == "*/*" {
		return true
	}

	if strings.HasSuffix(parsedType, "/*") {
		baseType := strings.TrimSuffix(parsedType, "/*")
		return strings.HasPrefix(supportedType, baseType+"/")
	}

	return false
}
