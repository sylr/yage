package utils

import "log"

func LogFatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	log.Printf("yage: error: "+format, v...)
	log.Fatalf("yage: report unexpected or unhelpful errors at https://github.com/sylr/yage/issues")
}

func Warningf(format string, v ...interface{}) {
	log.Printf("yage: warning: "+format, v...)
}

func ErrorWithHint(error string, hints ...string) {
	log.Printf("yage: error: %s", error)
	for _, hint := range hints {
		log.Printf("age: hint: %s", hint)
	}
	log.Fatalf("yage: report unexpected or unhelpful errors at https://github.com/sylr/yage/issues")
}
