package main

import (
	commontypes "github.com/criage-oss/criage-common/types"
)

// toCommonPackageManifest converts local PackageManifest to common/types.PackageManifest
func toCommonPackageManifest(pm *PackageManifest) *commontypes.PackageManifest {
	if pm == nil {
		return nil
	}
	return &commontypes.PackageManifest{
		Name:         pm.Name,
		Version:      pm.Version,
		Description:  pm.Description,
		Author:       pm.Author,
		License:      pm.License,
		Homepage:     pm.Homepage,
		Repository:   pm.Repository,
		Keywords:     append([]string(nil), pm.Keywords...),
		Dependencies: copyStringMap(pm.Dependencies),
		DevDeps:      copyStringMap(pm.DevDeps),
		Scripts:      copyStringMap(pm.Scripts),
		Files:        append([]string(nil), pm.Files...),
		Metadata:     copyAnyMap(pm.Metadata),
	}
}

// fromCommonPackageManifest converts common/types.PackageManifest to local PackageManifest
func fromCommonPackageManifest(pm *commontypes.PackageManifest) *PackageManifest {
	if pm == nil {
		return nil
	}
	return &PackageManifest{
		Name:         pm.Name,
		Version:      pm.Version,
		Description:  pm.Description,
		Author:       pm.Author,
		License:      pm.License,
		Homepage:     pm.Homepage,
		Repository:   pm.Repository,
		Keywords:     append([]string(nil), pm.Keywords...),
		Dependencies: copyStringMap(pm.Dependencies),
		DevDeps:      copyStringMap(pm.DevDeps),
		Files:        append([]string(nil), pm.Files...),
		Scripts:      copyStringMap(pm.Scripts),
		Metadata:     copyAnyMap(pm.Metadata),
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyAnyMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
