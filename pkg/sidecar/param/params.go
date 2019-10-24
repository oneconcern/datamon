// Copyright © 2018 One Concern
package param

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

var itemSep string
var kvSep string

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
	DestBundleID string `json:"destBundleID" yaml:"destBundleID"`
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
		_ struct{}
	} `json:"globalOpts" yaml:"globalOpts"`
	Bundles []fuseParamsBundleParams `json:"bundles" yaml:"bundles"`
	_ struct{}
}

// todo: ingestion of parameters as multiple environment variables

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
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	if fuseParams.Globals.Contributor.Name != "" || fuseParams.Globals.Contributor.Email != "" {
		if fuseParams.Globals.Contributor.Name == "" || fuseParams.Globals.Contributor.Email == "" {
			return rv, errors.New("contributor name and email must be set together or not at all")
		}
		rv, err = appendToParamString(rv, "e", fuseParams.Globals.Contributor.Email)
		if err != nil {
			return rv, fmt.Errorf("build parameter string: %v", err)
		}
		rv, err = appendToParamString(rv, "n", fuseParams.Globals.Contributor.Name)
		if err != nil {
			return rv, fmt.Errorf("build parameter string: %v", err)
		}
	}
	return strings.TrimSuffix(rv, itemSep), nil
}

func fuseParamsBundleString(bundleParams fuseParamsBundleParams) (string, error) {
	var err error
	rv := itemSep + kvSep
	rv, err = appendToParamString(rv, "sp", bundleParams.SrcPath)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "sr", bundleParams.SrcRepo)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "sl", bundleParams.SrcLabel)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "sb", bundleParams.SrcBundle)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "dp", bundleParams.DestPath)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "dr", bundleParams.DestRepo)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "dm", bundleParams.DestMessage)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "dl", bundleParams.DestLabel)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	rv, err = appendToParamString(rv, "dif", bundleParams.DestBundleID)
	if err != nil {
		return rv, fmt.Errorf("build parameter string: %v", err)
	}
	return strings.TrimSuffix(rv, itemSep), nil
}

func fuseParamsBundleStrings(fuseParams FUSEParams) (map[string]string, error) {
	rv := make(map[string]string)
	for _, bundleParams := range fuseParams.Bundles {
		bundleString, err := fuseParamsBundleString(bundleParams)
		if err != nil {
			return rv, fmt.Errorf("parameterize individual bundle: %v", err)
		}
		rv[bundleParams.Name] = bundleString
	}
	return rv, nil
}

// abstract serialization utility belt fn
func fieldsAsStringValues(iVal interface{}) ([]string, error) {
	v := reflect.ValueOf(iVal)
	kind := v.Kind()
	switch kind {
	case reflect.Bool:
		if iVal.(bool) {
			return []string{"true"}, nil
		} else {
			return []string{"false"}, nil
		}
	case reflect.String:
		return []string{iVal.(string)}, nil
	case reflect.Struct:
		rv := make([]string, 0)
		for i := 0; i < v.NumField(); i++ {
			nestedV := v.Field(i)
			vType := v.Type()
			nestedVStructField := vType.Field(i)
			if nestedVStructField.PkgPath != "" {
				continue
			}
			nestedIVal := nestedV.Interface()
			nestedStrings, err := fieldsAsStringValues(nestedIVal)
			if err != nil {
				return []string{}, err
			}
			rv = append(rv, nestedStrings...)
		}
		return rv, nil
	case reflect.Slice:
		rv := make([]string, 0)
		for i := 0; i < v.Len(); i++ {
			nestedV := v.Index(i)

			nestedIVal := nestedV.Interface()

			nestedStrings, err := fieldsAsStringValues(nestedIVal)
			if err != nil {
				return []string{}, err
			}
			rv = append(rv, nestedStrings...)
		}
		return rv, nil
	default:
		return []string{}, errors.New("unsupported kind")
	}
}

func stringToUniqRunes(str string) ([]rune, error) {
	rdr := strings.NewReader(str)
	rv := make([]rune, 0)
	for {
		ch, _, err := rdr.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		runeSeen := false
		for _, ech := range rv {
			if ech == ch {
				runeSeen = true
				break
			}
		}
		if !runeSeen {
			rv = append(rv, ch)
		}
	}
	return rv, nil
}

// set-ification with union operation
func mergeAndUniqifyRunes(runesArrs ...string) (string, error) {
	var rv strings.Builder
	for _, runesStr := range runesArrs {
		runes := make([]rune, 0)
		rdr := strings.NewReader(runesStr)
		for {
			ch, _, err := rdr.ReadRune()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			runes = append(runes, ch)
		}
		for _, runeNew := range runes {
			runeSeen := false
			rvrdr := strings.NewReader(rv.String())
 			for {
				runeExist, _, err := rvrdr.ReadRune()
				if err == io.EOF {
					break
				}
				if err != nil {
					return "", err
				}
				if runeNew == runeExist {
					runeSeen = true
					break
				}
			}
			if !runeSeen {
				_, err := rv.WriteRune(runeNew)
				if err != nil {
					return "", err
				}
			}
		}
	}
	return rv.String(), nil
}

// actually deterministic, despite name
func randCharNotInString(str string) (string, error) {
	runes, err := stringToUniqRunes(str)
	if err != nil {
		return "", err
	}
	addRune := '0'
	for {
		runeSeen := false
		for _, cr := range runes {
			if cr == addRune {
				runeSeen = true
				break
			}
		}
		if !runeSeen {
			break
		}
		addRune += 1
	}
	var rb strings.Builder
	_, err = rb.WriteRune(addRune)
	if err != nil {
		return "", err
	}
	return rb.String(), nil
}

func charsInFuseParams(fuseParams FUSEParams) (string, error) {
	stringVals, err := fieldsAsStringValues(fuseParams)
	if err != nil {
		return "", err
	}
	return mergeAndUniqifyRunes(stringVals...)
}

func setSeparators(fuseParams FUSEParams) error {
	var err error
	stringVals, err := fieldsAsStringValues(fuseParams)
	if err != nil {
		return err
	}
	invalidSeps, err := mergeAndUniqifyRunes(stringVals...)
	if err != nil {
		return err
	}
	itemSep, err = randCharNotInString(invalidSeps)
	if err != nil {
		return err
	}
	invalidSeps += itemSep
	kvSep, err = randCharNotInString(invalidSeps)
	if err != nil {
		return err
	}
	return nil
}

func FUSEParamsToEnvVars(fuseParams FUSEParams) (map[string]string, error) {
	rv := make(map[string]string)
	err := setSeparators(fuseParams)
	if err != nil {
		return rv, fmt.Errorf("bundles' parameters: %v", err)
	}
	bundleStrings, err := fuseParamsBundleStrings(fuseParams)
	if err != nil {
		return rv, fmt.Errorf("bundles' parameters: %v", err)
	}
	for bundleName, bundleString := range bundleStrings {
		rv[bundleEnvVarPrefix + bundleName] = bundleString
	}
	globalString, err := fuseParamsGlobalString(fuseParams)
	if err != nil {
		return rv, fmt.Errorf("global parameters: %v", err)
	}
	rv[fuseGlobalsEnvVar] = globalString
	return rv, nil
}

// todo: PG sidecar after FUSE sidecar
type PGParams struct {
	_                      struct{}
}

func init() {
	itemSep = ";"
	kvSep = ":"
}
