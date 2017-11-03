package env

import "testing"
import "os"
import "github.com/matryer/is"
import "time"
import "fmt"

type envVarOs struct {
	name  string
	value string
}

var testEnvVarsOs = []envVarOs{
	{"TEST_1", "VALUE_1"},
	{"TEST_2", "VALUE_2"},
	{"TEST_3", "VALUE_3"},
	{"TEST_4", "VALUE_4"},
	{"TEST_5", "VALUE_5"},
	{"TEST_6", "6"},
	{"TEST_TIME_1", "01/11/2017"}}

//Initial setup common for all test cases
func setup() {
	for i := 0; i < len(testEnvVarsOs); i++ {
		os.Setenv(testEnvVarsOs[i].name, testEnvVarsOs[i].value)
	}
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func TestLoadEnvVars_WithAllExportedVariables_ShouldSetVars(t *testing.T) {
	is := is.New(t)
	type Type1 struct {
		Var1 string    `env:"TEST_1"`
		Var2 string    `env:"TEST_2"`
		Var3 string    `env:"TEST_3"`
		Var4 string    `env:"TEST_4"`
		Var5 string    `env:"TEST_5"`
		Var6 int       `env:"TEST_6"`
		Var7 time.Time `env:"TEST_TIME_1" timeLayout:"02/01/2006"`
	}
	struct1 := Type1{}
	fmt.Println(LoadEnvVarsT(&struct1))
	expectedTime := time.Date(2017, time.November, 1, 0, 0, 0, 0, time.UTC)
	struct1_expected := Type1{"VALUE_1", "VALUE_2", "VALUE_3", "VALUE_4", "VALUE_5", 6, expectedTime}
	is.Equal(struct1, struct1_expected)
}

func TestLoadEnvVars_WithAllUnExportedVariables_ShouldNotSetVariables(t *testing.T) {
	is := is.New(t)
	type Type2 struct {
		var1 string `env:"TEST_1"`
		var2 string `env:"TEST_2"`
	}
	struct1 := Type2{}
	LoadEnvVars(&struct1)
	struct1_expected := Type2{}
	is.Equal(struct1, struct1_expected)
}

func TestLoadEnvVar_WithStringPtr_ShouldNotSetVariables(t *testing.T) {
	is := is.New(t)
	var var1 string
	LoadEnvVar(&var1, "TEST_1", "")
	expected := "VALUE_1"
	is.Equal(var1, expected)
}

func TestLoadEnvVar_WithIntPtr_ShouldNotSetVariables(t *testing.T) {
	is := is.New(t)
	var var1 int
	LoadEnvVar(&var1, "TEST_6", "")
	expected := 6
	is.Equal(var1, expected)
}

func TestLoadEnvVar_WithFloatPtrDefaultValue_ShouldNotSetVariables(t *testing.T) {
	is := is.New(t)
	var var1 float64
	LoadEnvVar(&var1, "TEST_7", "3.14")
	expected := 3.14
	is.Equal(var1, expected)
}

func TestLoadEnvVarT_WithTimePtr_ShouldNotSetVariables(t *testing.T) {
	is := is.New(t)
	var var1 time.Time
	LoadEnvVarT(&var1, "TEST_TIME_1", "", "02/01/2006")
	expected := time.Date(2017, time.November, 1, 0, 0, 0, 0, time.UTC)
	is.Equal(var1, expected)
}
