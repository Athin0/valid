package valid

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	arr := make([]string, len(v))
	for i, el := range v {
		arr[i] = el.Err.Error()
	}
	err := strings.Join(arr, "\n")
	return err
}

func Validate(v any) error {
	t := reflect.TypeOf(v)
	var validateErrs ValidationErrors

	err := ValidateType(t) //check struct it is, else return error
	if err != nil {
		return err
	}

	for i := 0; i < t.NumField(); i++ {
		fieldIType := t.Field(i)
		valueOfField := reflect.ValueOf(v).Field(i)

		s := fieldIType.Tag.Get("validate")
		if len(s) == 0 {
			continue
		}
		if !fieldIType.IsExported() {
			validateErrs = append(validateErrs, ValidationError{ErrValidateForUnexportedFields})
			continue
		}

		var keyVal []string
		keyVal, err = ParseTagVal(s)
		if err != nil {
			validateErrs = append(validateErrs, ValidationError{err})
			continue
		}
		keyTag, valTag := keyVal[0], keyVal[1]

		if valueOfField.Kind() == reflect.Slice {
			if slice, ok := valueOfField.Interface().([]int); ok {
				for _, el := range slice {
					err = ValidateElement(keyTag, valTag, el)
					if err != nil {
						validateErrs = append(validateErrs, ValidationError{err})
					}
				}
			}
			if slice, ok := valueOfField.Interface().([]string); ok {
				for _, el := range slice {
					err = ValidateElement(keyTag, valTag, el)
					if err != nil {
						validateErrs = append(validateErrs, ValidationError{err})
					}
				}
			}
			continue
		}

		var value interface{}
		switch valueOfField.Type().String() {
		case reflect.Int.String():
			value = int(valueOfField.Int())
		case reflect.String.String():
			value = valueOfField.String()
		}
		err = ValidateElement(keyTag, valTag, value)
		if err != nil {
			validateErrs = append(validateErrs, ValidationError{err})
		}

	}

	if len(validateErrs) != 0 {
		return validateErrs
	}
	return nil
}

func ValidateType(t reflect.Type) error {
	if t.Kind().String() != reflect.Struct.String() {
		return ErrNotStruct
	}
	return nil
}

func ParseTagVal(s string) ([]string, error) {
	arr := strings.Split(s, ":")
	if l := len(arr); l != 2 || arr[1] == "" {
		if arr[1] == "" {
			l = 1
		}
		return nil, fmt.Errorf("wrong number of arguments in tag: %s, len: %d, want 2", arr, l)
	}
	return arr, nil
}

func validLen(length string, a any) error {
	s, ok := a.(string)
	if !ok {
		return fmt.Errorf("validarion if len of string notfor string, for %t, %s", a, a)
	}

	if l, err := strconv.Atoi(length); err == nil {
		if len(s) != l {
			return fmt.Errorf("wrong length of %s, len: %d, want: %d", s, len(s), l)
		}
	} else {
		return ErrInvalidValidatorSyntax
	}

	return nil
}

func validIn(possibleValues string, s any) error {
	values := strings.Split(possibleValues, ",")
	switch s := s.(type) {
	case int:
		return validateInInt(s, values)
	case string:
		return validateInString(s, values)
	}
	return nil
}

func validateInInt(val int, values []string) error {
	ok := false
	for _, el := range values {
		v, err := strconv.Atoi(el)
		if err != nil {
			return ErrInvalidValidatorSyntax
		}
		if val == v {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("value %d not on range %s", val, values)
	}
	return nil
}
func validateInString(val string, values []string) error {
	ok := false
	for _, el := range values {
		if val == el {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("value %s not on range %s", val, values)
	}
	return nil
}

func validMinMax(minValue string, s any, less func(int, int) bool) error {
	min, err := strconv.Atoi(minValue)
	if err != nil {
		return ErrInvalidValidatorSyntax
	}
	switch s := s.(type) {
	case int:
		if less(s, min) {
			return fmt.Errorf("value %d less/more than min/max value - %d", s, min)
		}
	case string:
		if less(len(s), min) {
			return fmt.Errorf("value %s, len: %d less/more than min/max len - %d", s, len(s), min)
		}
	}
	return nil
}

func ValidateElement(keyTag, valTag string, value interface{}) error {
	var validateErr error
	switch keyTag {
	case "len":
		validateErr = validLen(valTag, value)
	case "in":
		validateErr = validIn(valTag, value)
	case "min":
		validateErr = validMinMax(valTag, value, func(a int, b int) bool {
			return a < b
		})
	case "max":
		validateErr = validMinMax(valTag, value, func(a int, b int) bool {
			return a > b
		})
	default:
		validateErr = fmt.Errorf("no such field tag: %s", keyTag)
	}

	if validateErr != nil {
		return validateErr
	}
	return nil
}
