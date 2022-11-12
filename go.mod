module sylr.dev/yage

go 1.19

require (
	filippo.io/age v1.0.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90
	golang.org/x/term v0.0.0-20220722155259-a9ba230a4035
	sylr.dev/yaml/age/v3 v3.0.0-20220527135827-28ffff5246ba
	sylr.dev/yaml/v3 v3.0.0-20220527135632-500fddf2b049
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20220909162455-aba9fc2a8ff2 // indirect
)

// TODO: remove this after https://github.com/sylr/go-yaml-age/pull/13 is merged
replace sylr.dev/yaml/age/v3 => github.com/WillAbides/go-yaml-age/v3 v3.0.0-20221112211920-5c10a0f209a6
