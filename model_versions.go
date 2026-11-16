package main

import "strings"

type modelVersionKey struct {
	family string
	suffix string
}

// retainLatestModels keeps the newest numeric version for each model family and variant.
func retainLatestModels(benchmarks []benchmark) {
	latest := make(map[modelVersionKey]string)
	for _, benchmark := range benchmarks {
		for _, point := range benchmark.Points {
			key, version, ok := modelVersionParts(point.Model)
			if !ok {
				continue
			}
			if current, exists := latest[key]; !exists || compareModelVersions(version, current) > 0 {
				latest[key] = version
			}
		}
	}
	if len(latest) == 0 {
		return
	}

	for i := range benchmarks {
		points := benchmarks[i].Points
		kept := 0
		for read := range points {
			point := points[read]
			key, version, ok := modelVersionParts(point.Model)
			if ok && compareModelVersions(version, latest[key]) != 0 {
				continue
			}
			if kept != read {
				points[kept] = point
			}
			kept++
		}
		clear(points[kept:])
		benchmarks[i].Points = points[:kept]
	}
}

func modelVersionParts(model string) (modelVersionKey, string, bool) {
	versionStart := -1
	for i := range len(model) {
		if model[i] >= '0' && model[i] <= '9' {
			versionStart = i
			break
		}
	}
	if versionStart < 0 {
		return modelVersionKey{}, "", false
	}

	versionEnd := versionStart
	for versionEnd < len(model) {
		if model[versionEnd] >= '0' && model[versionEnd] <= '9' {
			versionEnd++
			continue
		}
		if model[versionEnd] == '.' && versionEnd+1 < len(model) && model[versionEnd+1] >= '0' && model[versionEnd+1] <= '9' {
			versionEnd++
			continue
		}
		break
	}
	return modelVersionKey{
		family: canonicalModel(model[:versionStart]),
		suffix: canonicalModel(model[versionEnd:]),
	}, model[versionStart:versionEnd], true
}

func compareModelVersions(a, b string) int {
	for a != "" || b != "" {
		aPart, bPart := "0", "0"
		if a != "" {
			aPart, a = nextVersionComponent(a)
		}
		if b != "" {
			bPart, b = nextVersionComponent(b)
		}
		aPart = strings.TrimLeft(aPart, "0")
		bPart = strings.TrimLeft(bPart, "0")
		if len(aPart) != len(bPart) {
			if len(aPart) < len(bPart) {
				return -1
			}
			return 1
		}
		if aPart != bPart {
			if aPart < bPart {
				return -1
			}
			return 1
		}
	}
	return 0
}

func nextVersionComponent(version string) (string, string) {
	if separator := strings.IndexByte(version, '.'); separator >= 0 {
		return version[:separator], version[separator+1:]
	}
	return version, ""
}
