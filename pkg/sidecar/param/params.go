// Copyright © 2018 One Concern
package param

import (
	"errors"
	"strings"
)

const (
	itemSep = ";"
	kvSep = ":"
)

const (
	fuseGlobalsEnvVar = "dm_fuse_opts"
	bundleEnvVarPrefix = "dm_fuse_bd_"
)

func containsSep(val string) bool {
	return strings.Contains(val, itemSep) || strings.Contains(val, kvSep)
}

type fuseParamsBundleParams struct{
	Name string `json:"name" yaml:"name"`
	SrcPath string `json:"srcPath" yaml:"srcPath"`
	SrcRepo string `json:"srcRepo" yaml:"srcRepo"`
	SrcLabel string `json:"srcLabel" yaml:"srcLabel"`
	SrcBundle string `json:"srcBundle" yaml:"srcBundle"`
	DestPath string `json:"destPath" yaml:"destPath"`
	DestRepo string `json:"destRepo" yaml:"destRepo"`
	DestMessage string `json:"destMessage" yaml:"destMessage"`
	DestLabel string `json:"destLabel" yaml:"destLabel"`
	_ struct{}
}

type FUSEParams struct {
	Globals struct{
		SleepInsteadOfExit bool `json:"sleepInsteadOfExit" yaml:"sleepInsteadOfExit"`
		CoordPoint string `json:"coordPoint" yaml:"coordPoint"`
		Contributor struct{
			Name string `json:"name" yaml:"name"`
			Email string `json:"email" yaml:"email"`
			_ struct{}
		} `json:"contributor" yaml:"contributor"`
		// todo: this was *never* a good idea [as a global parameter]...
		BundleID string `json:"bundleID" yaml:"bundleID"`
		_ struct{}
	} `json:"globalOpts" yaml:"globalOpts"`
	Bundles []fuseParamsBundleParams `json:"bundles" yaml:"bundles"`
	_ struct{}
}

// todo: serialize to .conf file(s)

// todo: chain errors with fmt.Errorf() throughout

// todo: dynamic separators to allow arbitrary values not covering entire unicode plane
func appendToParamString(paramString string, paramName string, paramVal string) (string, error) {
	if paramVal == "" {
		return paramString, nil
	}
	if containsSep(paramVal) {
		return paramString, errors.New("variables may not contain separator values")
	}
	return paramString + paramName + kvSep + paramVal + itemSep, nil
}

func fuseParamsGlobalString(fuseParams FUSEParams) (string, error) {
	var err error
	rv := itemSep + kvSep
	if fuseParams.Globals.SleepInsteadOfExit {
		rv += "S" + itemSep
	}
	if fuseParams.Globals.CoordPoint == "" {
		return rv, errors.New("coordination point not set")
	}
	rv, err = appendToParamString(rv, "c", fuseParams.Globals.CoordPoint)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "i", fuseParams.Globals.BundleID)
	if err != nil {
		return rv, err
	}
	if fuseParams.Globals.Contributor.Name != "" || fuseParams.Globals.Contributor.Email != "" {
		if fuseParams.Globals.Contributor.Name == "" || fuseParams.Globals.Contributor.Email == "" {
			return rv, errors.New("contributor name and email must be set together or not at all")
		}
		rv, err = appendToParamString(rv, "e", fuseParams.Globals.Contributor.Email)
		if err != nil {
			return rv, err
		}
		rv, err = appendToParamString(rv, "n", fuseParams.Globals.Contributor.Name)
		if err != nil {
			return rv, err
		}
	}
	return strings.TrimSuffix(rv, itemSep), nil
}

func fuseParamsBundleString(bundleParams fuseParamsBundleParams) (string, error) {
	var err error
	rv := itemSep + kvSep
	rv, err = appendToParamString(rv, "sp", bundleParams.SrcPath)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "sr", bundleParams.SrcRepo)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "sl", bundleParams.SrcLabel)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "sb", bundleParams.SrcBundle)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "dp", bundleParams.DestPath)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "dr", bundleParams.DestRepo)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "dm", bundleParams.DestMessage)
	if err != nil {
		return rv, err
	}
	rv, err = appendToParamString(rv, "dl", bundleParams.DestLabel)
	if err != nil {
		return rv, err
	}
	return strings.TrimSuffix(rv, itemSep), nil
}

func fuseParamsBundleStrings(fuseParams FUSEParams) (map[string]string, error) {
	rv := make(map[string]string)
	for _, bundleParams := range fuseParams.Bundles {
		bundleString, err := fuseParamsBundleString(bundleParams)
		if err != nil {
			return rv, err
		}
		rv[bundleParams.Name] = bundleString
	}
	return rv, nil
}

func FUSEParamsToEnvVars(fuseParams FUSEParams) (map[string]string, error) {
	rv := make(map[string]string)
	bundleStrings, err := fuseParamsBundleStrings(fuseParams)
	if err != nil {
		return rv, err
	}
	for bundleName, bundleString := range bundleStrings {
		rv[bundleEnvVarPrefix + bundleName] = bundleString
	}
	globalString, err := fuseParamsGlobalString(fuseParams)
	if err != nil {
		return rv, err
	}
	rv[fuseGlobalsEnvVar] = globalString
	return rv, nil
}

/*
example of marshal/unmarshal using json/yaml
	var bundleDescriptorBuffer []byte
	err = yaml.Unmarshal(bundleDescriptorBuffer, &bundle.BundleDescriptor)
	buffer, err := yaml.Marshal(bundle.BundleDescriptor)
*/


type PGParams struct {
	_                      struct{}
}
