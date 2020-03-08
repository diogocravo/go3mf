package production

import (
	"sort"

	"github.com/qmuntal/go3mf"
	specerr "github.com/qmuntal/go3mf/errors"
)

// Validate checks that the model is conformant with the 3MF spec.
// Core spec related checks are not reported.
func Validate(model *go3mf.Model) []error {
	var hasExt bool
	for _, ext := range model.Namespaces {
		if ext.Space == ExtensionName {
			hasExt = true
			break
		}
	}
	if !hasExt {
		return nil
	}
	err := make([]error, 0)
	err = validateChilds(model, err)
	return validateRoot(model, err)
}

func validateObjects(path string, isRoot bool, objs []*go3mf.Object, err []error) (bool, []error) {
	const attrComponent = "component"
	var (
		extU        *UUID
		extP        *PathUUID
		mustRequire bool
	)
	for i, obj := range objs {
		if ok := obj.ExtensionAttr.Get(&extU); ok && extU != nil {
			if validateUUID(string(*extU)) != nil {
				err = append(err, specerr.NewObject(path, i, specerr.ErrUUID))
			}
		} else {
			err = append(err, specerr.NewObject(path, i, &specerr.MissingFieldError{Name: attrProdUUID}))
		}
		for j, comp := range obj.Components {
			if comp.ExtensionAttr.Get(&extP) {
				if extP.UUID == "" {
					err = append(err, specerr.NewObject(path, i, &specerr.IndexedError{Name: attrComponent, Index: j, Err: &specerr.MissingFieldError{Name: attrProdUUID}}))
				} else if validateUUID(string(extP.UUID)) != nil {
					err = append(err, specerr.NewObject(path, i, &specerr.IndexedError{Name: attrComponent, Index: j, Err: specerr.ErrUUID}))
				}
				if extP.Path != "" && extP.Path != path {
					if isRoot {
						// Path is validated as part if the core validations
						mustRequire = true
					} else {
						err = append(err, specerr.NewObject(path, i, &specerr.IndexedError{Name: attrComponent, Index: j, Err: specerr.ErrProdRefInNonRoot}))
					}
				}
			} else {
				err = append(err, specerr.NewObject(path, i, &specerr.IndexedError{Name: attrComponent, Index: j, Err: &specerr.MissingFieldError{Name: attrProdUUID}}))
			}
		}
	}
	return mustRequire, err
}

func validateChilds(model *go3mf.Model, err []error) []error {
	s := make([]string, 0, len(model.Childs))
	for path := range model.Childs {
		s = append(s, path)
	}
	sort.Strings(s)
	for _, path := range s {
		c := model.Childs[path]
		_, err = validateObjects(path, false, c.Resources.Objects, err)
	}
	return err
}

func validateRoot(model *go3mf.Model, err []error) []error {
	var mustRequire bool
	path := model.PathOrDefault()
	mustRequire, err = validateObjects(path, true, model.Resources.Objects, err)
	var extU *UUID
	if ok := model.Build.ExtensionAttr.Get(&extU); ok && extU != nil {
		if validateUUID(string(*extU)) != nil {
			err = append(err, &specerr.BuildError{Err: specerr.ErrUUID})
		}
	} else {
		err = append(err, &specerr.BuildError{Err: &specerr.MissingFieldError{Name: attrProdUUID}})
	}
	var ext *PathUUID
	for i, item := range model.Build.Items {
		if item.ExtensionAttr.Get(&ext) {
			if ext.UUID == "" {
				err = append(err, specerr.NewItem(i, &specerr.MissingFieldError{Name: attrProdUUID}))
			} else if validateUUID(string(ext.UUID)) != nil {
				err = append(err, specerr.NewItem(i, specerr.ErrUUID))
			}
			if ext.Path != "" && ext.Path != model.Path {
				// Path is validated as part if the core validations
				mustRequire = true
			}
		} else {
			err = append(err, specerr.NewItem(i, &specerr.MissingFieldError{Name: attrProdUUID}))
		}
	}
	if mustRequire {
		var extRequired bool
		for _, r := range model.RequiredExtensions {
			if r == ExtensionName {
				extRequired = true
				break
			}
		}
		if !extRequired {
			err = append(err, specerr.ErrProdExtRequired)
		}
	}
	return err
}
