// Copyright © 2018 One Concern
package param

import (
	"errors"
)

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

type pgParamsDBParams struct{
	Name string `json:"name" yaml:"name"`
	Port int `json:"pgPort" yaml:"pgPort"`
	DestRepo string `json:"destRepo" yaml:"destRepo"`
	DestMessage string `json:"destMessage" yaml:"destMessage"`
	DestLabel string `json:"destLabel" yaml:"destLabel"`
	SrcRepo string `json:"srcRepo" yaml:"srcRepo"`
	SrcLabel string `json:"srcLabel" yaml:"srcLabel"`
	SrcBundle string `json:"srcBundle" yaml:"srcBundle"`
	_ struct{}
}

type PGParams struct {
	Globals struct{
		SleepInsteadOfExit bool `json:"sleepInsteadOfExit" yaml:"sleepInsteadOfExit"`
		IgnorePGVersionMismatch bool `json:"ignorePGVersionMismatch" yaml:"ignorePGVersionMismatch"`
		CoordPoint string `json:"coordPoint" yaml:"coordPoint"`
		Contributor struct{
			Name string `json:"name" yaml:"name"`
			Email string `json:"email" yaml:"email"`
			_ struct{}
		} `json:"contributor" yaml:"contributor"`
		_ struct{}
	} `json:"globalOpts" yaml:"globalOpts"`
	Databases []pgParamsDBParams `json:"databases" yaml:"databases"`
	_ struct{}
}

type FUSEParamsOption func(fuseParams *FUSEParams)

func FUSECoordPoint(coordPoint string) FUSEParamsOption {
	return func(fuseParams *FUSEParams) {
		fuseParams.Globals.CoordPoint = coordPoint
	}
}

func FUSEContributor(name string, email string) FUSEParamsOption {
	return func(fuseParams *FUSEParams) {
		fuseParams.Globals.Contributor.Name = name
		fuseParams.Globals.Contributor.Email = email
	}
}

func NewFUSEParams(fuseOpts ...FUSEParamsOption) (FUSEParams, error) {
	fuseParams := FUSEParams{}
	for _, apply := range fuseOpts {
		apply(&fuseParams)
	}
	if fuseParams.Globals.CoordPoint == "" {
		return fuseParams, errors.New("coordination point not set")
	}

	return fuseParams, nil
}

type FUSEParamsBDOption func(bdParams *fuseParamsBundleParams)

func BDName(name string) FUSEParamsBDOption {
	return func(bdParams *fuseParamsBundleParams) {
		bdParams.Name = name
	}
}

func BDSrcByLabel(path string, repo string, label string) FUSEParamsBDOption {
	return func(bdParams *fuseParamsBundleParams) {
		bdParams.SrcPath = path
		bdParams.SrcRepo = repo
		bdParams.SrcLabel = label
	}
}

func BDSrcByBundleID(path string, repo string, bundle string) FUSEParamsBDOption {
	return func(bdParams *fuseParamsBundleParams) {
		bdParams.SrcPath = path
		bdParams.SrcRepo = repo
		bdParams.SrcBundle = bundle
	}
}

func BDDest(repo string, msg string) FUSEParamsBDOption {
	return func(bdParams *fuseParamsBundleParams) {
		bdParams.DestRepo = repo
		bdParams.DestMessage = msg
	}
}

func BDDestLabel(label string) FUSEParamsBDOption {
	return func(bdParams *fuseParamsBundleParams) {
		bdParams.DestLabel = label
	}
}

func BDDestBundleIDFile(bundleIDFile string) FUSEParamsBDOption {
	return func(bdParams *fuseParamsBundleParams) {
		bdParams.DestBundleID = bundleIDFile
	}
}

func (fuseParams *FUSEParams) AddBundle(bdOpts ...FUSEParamsBDOption) error {
	bdParams := fuseParamsBundleParams{}
	for _, apply := range bdOpts {
		apply(&bdParams)
	}
	if bdParams.Name == "" {
		return errors.New("bundle name not set")
	}
	if bdParams.SrcLabel != "" && bdParams.SrcBundle != "" {
		return errors.New("label and bundle id are mutually exclusive")
	}
	destIsSet := bdParams.DestRepo != "" && bdParams.DestMessage != ""
	if bdParams.DestLabel != "" && !destIsSet {
		return errors.New("destination label setting requires destination being set")
	}
	if bdParams.DestBundleID != "" && !destIsSet {
		return errors.New("destination bundle id file setting requires destination being set")
	}
	fuseParams.Bundles = append(fuseParams.Bundles, bdParams)
	return nil
}

type PGParamsOption func(pgParams *PGParams)

func PGCoordPoint(coordPoint string) PGParamsOption {
	return func(pgParams *PGParams) {
		pgParams.Globals.CoordPoint = coordPoint
	}
}

func PGContributor(name string, email string) PGParamsOption {
	return func(pgParams *PGParams) {
		pgParams.Globals.Contributor.Name = name
		pgParams.Globals.Contributor.Email = email
	}
}

func NewPGParams(pgOpts ...PGParamsOption) (PGParams, error) {
	pgParams := PGParams{}
	for _, apply := range pgOpts {
		apply(&pgParams)
	}
	if pgParams.Globals.CoordPoint == "" {
		return pgParams, errors.New("coordination point not set")
	}
	return pgParams, nil
}

type PGParamsDBOption func(dbParams *pgParamsDBParams)

func DBNameAndPort(name string, port int) PGParamsDBOption {
	return func(dbParams *pgParamsDBParams) {
		dbParams.Name = name
		dbParams.Port = port
	}
}

func DBDest(repo string, message string) PGParamsDBOption {
	return func(dbParams *pgParamsDBParams) {
		dbParams.DestRepo = repo
		dbParams.DestMessage = message
	}
}

func DBDestLabel(label string) PGParamsDBOption {
	return func(dbParams *pgParamsDBParams) {
		dbParams.DestLabel = label
	}
}

func DBSrcByLabel(repo string, label string) PGParamsDBOption {
	return func(dbParams *pgParamsDBParams) {
		dbParams.SrcRepo = repo
		dbParams.SrcLabel = label
	}
}

func DBSrcByBundle(repo string, bundle string) PGParamsDBOption {
	return func(dbParams *pgParamsDBParams) {
		dbParams.SrcRepo = repo
		dbParams.SrcBundle = bundle
	}
}

func (pgParams *PGParams) AddDatabase(dbOpts ...PGParamsDBOption) error {
	dbParams := pgParamsDBParams{}
	for _, apply := range dbOpts {
		apply(&dbParams)
	}
	if dbParams.Name == "" {
		return errors.New("database name not set")
	}
	if dbParams.Port == 0 {
		return errors.New("database port not set")
	}
	if dbParams.DestRepo == "" || dbParams.DestMessage == "" {
		return errors.New("database destination not set")
	}
	if dbParams.SrcLabel != "" && dbParams.SrcBundle != "" {
		return errors.New("specifying source by bundle and label is mutually exclusive")
	}
	pgParams.Databases = append(pgParams.Databases, dbParams)
	return nil
}
