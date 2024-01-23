package main

func ExampleEventBasic() {
	app.Run([]string{"nak", "event", "--ts", "1699485669"})
	// Output:
	// {"id":"8c848702541668af204bcbbb4d7a760fef0d936d946c1260eaba0d8074ca12ac","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1699485669,"kind":1,"tags":[],"content":"hello from the nostr army knife","sig":"26e20e69605f08ca7d3e6768ff40232eda31fb550dd7fd4158fa33ad060b1a062c2b6409059374f1ad32867ac0d37581d9cae70ceac08cbc599d9462c9b254ec"}
}

func ExampleEventComplex() {
	app.Run([]string{"nak", "event", "--ts", "1699485669", "-k", "11", "-c", "skjdbaskd", "--sec", "17", "-t", "t=spam", "-e", "36d88cf5fcc449f2390a424907023eda7a74278120eebab8d02797cd92e7e29c", "-t", "r=https://abc.def?name=foobar;nothing"})
	// Output:
	// {"id":"33159e38e02a903478b7189adc528d4f478c7231ce8c73031f7ee4787327227b","pubkey":"2fa2104d6b38d11b0230010559879124e42ab8dfeff5ff29dc9cdadd4ecacc3f","created_at":1699485669,"kind":11,"tags":[["t","spam"],["r","https://abc.def?name=foobar","nothing"],["e","36d88cf5fcc449f2390a424907023eda7a74278120eebab8d02797cd92e7e29c"]],"content":"skjdbaskd","sig":"be36888db7b3effc79a27d04837fb25be008d5a2c0b1c0642102dda1631c499d9a9ba176d713554c156723d8de5e72eee51099990ebcf62ff0f84b4b31462699"}
}

func ExampleReq() {
	app.Run([]string{"nak", "req", "-k", "1", "-l", "18", "-a", "2fa2104d6b38d11b0230010559879124e42ab8dfeff5ff29dc9cdadd4ecacc3f", "-e", "aec4de6d051a7c2b6ca2d087903d42051a31e07fb742f1240970084822de10a6"})
	// Output:
	// ["REQ","nak",{"kinds":[1],"authors":["2fa2104d6b38d11b0230010559879124e42ab8dfeff5ff29dc9cdadd4ecacc3f"],"#e":["aec4de6d051a7c2b6ca2d087903d42051a31e07fb742f1240970084822de10a6"],"limit":18}]
}

func ExampleEncodeNpub() {
	app.Run([]string{"nak", "encode", "npub", "a6a67ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f8179822"})
	// Output:
	// npub156n8a7wuhwk9tgrzjh8gwzc8q2dlekedec5djk0js9d3d7qhnq3qjpdq28
}

func ExampleEncodeNprofile() {
	app.Run([]string{"nak", "encode", "nprofile", "-r", "wss://example.com", "a6a67ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f8179822"})
	// Output:
	// nprofile1qqs2dfn7l8wthtz45p3ftn58pvrs9xlumvkuu2xet8egzkcklqtesgspz9mhxue69uhk27rpd4cxcefwvdhk6fl5jug
}
