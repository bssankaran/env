package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	ENV         = "env"
	TIME_LAYOUT = "timeLayout"
	SEP         = ","
)

type fieldData struct {
	envVarName string
	defaultVal string
	strVal     string
	timeLayout string
	field      reflect.Value
}

type FieldErrorCode int

const (
	ENV_VAR_NOT_FOUND FieldErrorCode = 1 << iota
	ENV_VAR_PARSING_ERROR
	DEFAULT_VALUE_NOT_SPECIFIED
	DEFAULT_VALUE_PARSING_ERROR
	NIL FieldErrorCode = 0
)

// This is an internal type used inside EnvError
type FieldError struct {
	FieldName string         // the errored field name
	ErrorCode FieldErrorCode // the combined error code
	ErrorMsg  string         // the combined error messages
}

func (e FieldError) AddError(errorCode FieldErrorCode, errorMsg string) {
	e.ErrorCode += errorCode
	e.ErrorMsg = fmt.Sprintf("%s\n\n%s", errorMsg, e.ErrorMsg)
}

func (e FieldError) Error() string {
	return fmt.Sprintf("Field Name : %s \nError:\n%s\n\n", e.FieldName, e.ErrorMsg)
}

// An EnvError records all the failed conversions of environment variables
type StructError struct {
	FieldErrors []FieldError
	ErrorMsg    string
}

func (e StructError) Error() string {
	var errors []string
	for _, fieldError := range e.FieldErrors {
		errors = append(errors, fieldError.Error())
	}
	e.ErrorMsg = strings.Join(errors, "\n")
	return e.ErrorMsg
}

//LoadEnvVars takes a pointer to a struct and loads the
// values of each of its variables taged `env` from the
// respective environment variable of the underlying OS.
//The name of the environment variable corresponding to
// a struct variable is determined from its StructTag
// named 'env'.
func LoadEnvVars(structPtr interface{}) error {
	return loadEnvVars(structPtr, false, false)
}

func LoadEnvVarsF(structPtr interface{}) error {
	return loadEnvVars(structPtr, true, false)
}

func LoadEnvVarsT(structPtr interface{}) error {
	return loadEnvVars(structPtr, false, true)
}

func LoadEnvVarsTF(structPtr interface{}) error {
	return loadEnvVars(structPtr, true, true)
}

func LoadEnvVar(ptr interface{}, envVarName, defaultVal string) error {
	return loadEnvVar(ptr, envVarName, defaultVal, false, "")
}

func LoadEnvVarT(ptr interface{}, envVarName, defaultVal string, timeLayout string) error {
	return loadEnvVar(ptr, envVarName, defaultVal, true, timeLayout)
}

func loadEnvVar(ptr interface{}, envVarName, defaultVal string, withTime bool, timeLayout string) error {
	_fieldData := fieldData{envVarName, defaultVal, "", timeLayout, reflect.ValueOf(ptr).Elem()}
	_fieldError := validateAndSet(&_fieldData, withTime)
	if _fieldError != (FieldError{}) {
		_fieldError.FieldName = envVarName
		return _fieldError
	}
	return nil
}

func loadEnvVars(structPtr interface{}, force bool, withTime bool) error {
	err := StructError{}
	value := reflect.ValueOf(structPtr).Elem()
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.Type().NumField(); i++ {
			//For each field in the struct marked with `env` tag,
			// get the value of corresponding environment variable for each
			// field from the os and set it in the struct. If a field is
			// not marked with `env` tag, the field is ignored.
			envVar := strings.SplitN(value.Type().Field(i).Tag.Get(ENV), SEP, 2)
			_fieldData := fieldData{}
			if len(envVar) >= 1 {
				_fieldData.envVarName = envVar[0]
			}
			if len(envVar) >= 2 {
				_fieldData.defaultVal = envVar[1]
			}
			if (_fieldData.envVarName == "") && force {
				_fieldData.envVarName = value.Type().Field(i).Name
			}
			if withTime {
				_fieldData.timeLayout = value.Type().Field(i).Tag.Get(TIME_LAYOUT)
			}
			_fieldData.field = value.Field(i)
			if _fieldData.envVarName != "" {
				_fieldError := validateAndSet(&_fieldData, withTime)
				if _fieldError != (FieldError{}) {
					_fieldError.FieldName = value.Type().Field(i).Name
					err.FieldErrors = append(err.FieldErrors, _fieldError)
				}
			}
		}
	}
	if err.FieldErrors != nil {
		return err
	}
	return nil
}

//validateAndSet checks whether the field is valid and is changable
// and sets its value only if the above condition is true. Else, it ignores the field.
//An error is returned if the environment variable is not found or if there are any parsing errors.
func validateAndSet(_fieldData *fieldData, withTime bool) FieldError {
	if _fieldData.field.IsValid() {
		if _fieldData.field.CanSet() {
			switch _fieldData.field.Kind() {
			case reflect.String:
				return setField(_fieldData, setStrField)
			case reflect.Int:
				return setField(_fieldData, setIntField)
			case reflect.Float64:
				return setField(_fieldData, setFloatField)
			case reflect.Bool:
				return setField(_fieldData, setBoolField)
			}
			if withTime && (_fieldData.field.Type() == reflect.TypeOf(time.Time{})) {
				return setField(_fieldData, setTimeField)
			}
		}
	}
	return FieldError{}
}

// setField uses the function setFunc to set various types fo values to fieldValue
// It handles the various possible error conditions and returns a fieldError instance
func setField(_fieldData *fieldData, setFunc func(*fieldData, string) error) FieldError {
	_fieldData.strVal = os.Getenv(_fieldData.envVarName)
	fe := FieldError{}
	var err error
	if _fieldData.strVal != "" {
		err = setFunc(_fieldData, _fieldData.strVal)
		if err == nil {
			return fe
		}
		fe.AddError(ENV_VAR_PARSING_ERROR, fmt.Sprintf("%s\n%s", "Error parsing env variable.Trying to set default value...", err.Error()))
	} else {
		fe.AddError(ENV_VAR_NOT_FOUND, "Env Variable not found. Trying to set default value")
	}
	if _fieldData.defaultVal != "" {
		err = setFunc(_fieldData, _fieldData.defaultVal)
		if err == nil {
			return fe
		}
		fe.AddError(DEFAULT_VALUE_PARSING_ERROR, fmt.Sprintf("%s\n%s", "Error while parsing the default value. Ignoring the field...", err.Error()))
		return fe
	}
	fe.AddError(DEFAULT_VALUE_NOT_SPECIFIED, "Default value not specified. Ignoring the field.")
	return fe
}

//setStrField sets a string value to the field assuming that the canSet()
// for the field is true.
func setStrField(_fieldData *fieldData, valueToSet string) error {
	_fieldData.field.SetString(valueToSet)
	return nil
}

//setIntField sets an int value to the field assuming that the canSet()
// for the field is true.
//The function parses the string to integer and returns any error
// incurred while parsing the field value
func setIntField(_fieldData *fieldData, valueToSet string) error {
	intVal, err := strconv.Atoi(valueToSet)
	if err != nil {
		return err
	}
	_fieldData.field.SetInt(int64(intVal))
	return nil
}

//setFloatField sets a float value to the field assuming that the canSet()
// for the field is true.
//The function parses the string to float and returns any error
// incurred while parsing the field value
func setFloatField(_fieldData *fieldData, valueToSet string) error {
	floatVal, err := strconv.ParseFloat(valueToSet, 64)
	if err != nil {
		return err
	}
	_fieldData.field.SetFloat(floatVal)
	return nil
}

//setBoolField sets a boolean value to the field assuming that the canSet()
// for the field is true.
//The function parses the string to boolean and returns any error
// incurred while parsing the field value
func setBoolField(_fieldData *fieldData, valueToSet string) error {
	boolVal, err := strconv.ParseBool(valueToSet)
	if err != nil {
		return err
	}
	_fieldData.field.SetBool(boolVal)
	return nil
}

//setTimeField sets a time.Time value to the field assuming that the canSet()
// for the field is true.
//The function parses the string to time.Time and returns any error
// incurred while parsing the field value
func setTimeField(_fieldData *fieldData, valueToSet string) error {
	timeVal, err := time.Parse(_fieldData.timeLayout, valueToSet)
	if err != nil {
		return err
	}
	_fieldData.field.Set(reflect.ValueOf(timeVal))
	return nil
}
