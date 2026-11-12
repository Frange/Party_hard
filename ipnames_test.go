package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Owner's policy (2026-11-11): visible names of real people and of characters/brands of existing works
// carry one letter changed, and proper names are never translated (ipnames.go). The ORIGINAL names are
// kept here only as FNV-1a hashes of their normalised form (lower case, no accents, letters and digits
// joined without spaces), so this test does not publish them again. To add one: hash the normalised
// name with ipHash and put it in the list (ipOriginalInText also scans free text, so only names that
// are not ordinary words go there).

func ipHashSet(hs ...uint64) map[uint64]bool {
	m := map[uint64]bool{}
	for _, h := range hs {
		m[h] = true
	}
	return m
}

var ipOriginalNames = ipHashSet(
	0x003d01d762fa241c, 0x005c2ef513cd35cd, 0x02ce0fa1f9e4c1ef, 0x047c5b9a22f45728, 0x05ec1e91ad8fb6c4,
	0x061dc04c9eeec931, 0x0650c3e541e645e0, 0x067880d1317f6aca, 0x070eceb46ad585c4, 0x088e3e07b53950dc,
	0x09603518bf7ce1f8, 0x09cdc7ab1d0017bf, 0x0ac3ad5ea5ec0f33, 0x0b65881e26fa612c, 0x0c90ae3261f7f7d1,
	0x0e6cbc706117a669, 0x0f593bbb8ffc84bd, 0x0f6f9771462c7f72, 0x1079d91db1b62457, 0x109dcb7ba0977f5d,
	0x120f603712e1c944, 0x12ead608153db536, 0x13875ee2f0a9ef12, 0x14a82b6b2962e1f5, 0x14b22e0b6b9998f9,
	0x15b594dfc114a1b0, 0x15cb117198971824, 0x15fc478071cef8bd, 0x164d81059d3ee45a, 0x167976e8ee6e04bd,
	0x169ec3e28995ef8c, 0x17cbb4a98618cd49, 0x1852a0eb2ffe4278, 0x18a08167625327f6, 0x19a37b7afe48ff9c,
	0x1a18ab744e1da2a4, 0x1ceb9566f5581bba, 0x1e305f9bff4a5f26, 0x1e86b5ad470e9eba, 0x1ed680fbed26c5df,
	0x1ee87030d3733398, 0x1ef1130a2a591b98, 0x20a196714caf0f43, 0x20a46fa8d6e51a56, 0x2138dd192571c4c9,
	0x22be4862ff4f1779, 0x23a4527fbd0fbd36, 0x245039d03398b6e7, 0x24928dada1eb5068, 0x24f648c7d94c4f06,
	0x252208f17659c64c, 0x258f998d86d2d610, 0x259bbbef3259df98, 0x27ab4f4c93847fac, 0x2857a0e10d7bafef,
	0x28974a678c45bbaa, 0x29ca2ce867d1c321, 0x2a6dd11d7d45b952, 0x2b05263185d9d841, 0x2b982e7349e2fa32,
	0x2ba7bb672cd82f0e, 0x2ce644b2848b9571, 0x2ce88f6108bf142f, 0x2d179517845469c8, 0x2d49ecaa4db67e33,
	0x2d5d43dbf82cb3f8, 0x2d8e78ab6018a527, 0x2e3ffd7127fd7f08, 0x2e7d90ced09d6ad2, 0x305c5b3824233bc0,
	0x31ea565ee7368cfa, 0x31fa1801a737b103, 0x32cee4c2ab70a880, 0x34f43e64be50d94b, 0x357e58737e0837b0,
	0x396d9446f27ea4a3, 0x39848c84669eb3c9, 0x399b4f5b394a6dee, 0x3a5fc99af05819fa, 0x3bd7bc8f38944cb6,
	0x3d1eaf7d15fe24e6, 0x3de9df995d5e33b1, 0x3dffa47537ea2280, 0x3fef3158c3ca79e0, 0x40642f0a2d060b5e,
	0x406e0f10063a3d19, 0x40cc9e804c423a3c, 0x4221256a2a0a57ce, 0x423d69afe058c046, 0x42a89261492ba8fb,
	0x42bf70cce58211d4, 0x42f91688ba6ad7a1, 0x42fe75330d39ea85, 0x432ce3e6fcd2ddc0, 0x43634fcfc7b4dbb0,
	0x44e8a3e98842f0fc, 0x45bd50288b392da3, 0x47363c46849812ca, 0x476496930d77b32e, 0x47e86330c50f4b02,
	0x47fe4dc823bba0c3, 0x4845ad434f632991, 0x48ebfc907b503538, 0x493d27435a2ceff1, 0x4aa9b511ae682234,
	0x4b53d6abbc677791, 0x4c30a8ae846dc147, 0x4c66c01daa7e3384, 0x4e025cbe7c291773, 0x4e349fccb4cc5414,
	0x51027acfec775d0e, 0x51c0714669443d7a, 0x5220203696d77840, 0x53491d5d62b3c8ff, 0x53ad97566ebb444f,
	0x544eb2e43014371a, 0x54dae658e563facc, 0x5505e40c929c0d49, 0x558c468abcc5e759, 0x570dcfe3fe703f89,
	0x571b8bd61f48a52e, 0x58ae8e2dd988d3fc, 0x58e7c122560aac12, 0x5b5e3806c74d8ac7, 0x5b81dff7e4c70508,
	0x5c56bcf7f111dca8, 0x5c5779aff3a6449d, 0x5f20b61e90715cea, 0x5f52735fe96eb3c5, 0x60b82128fcc4d60a,
	0x626336716aee23b1, 0x63b7b58aa0cae447, 0x63e84a04a06080f1, 0x643be369f69d3f4d, 0x665cf190bf59a282,
	0x66cf0f55c0b15115, 0x6867107ceb11b09f, 0x6a9b423083c75547, 0x6b46a0c1e0c88f14, 0x6c4dd05c92d19ae8,
	0x6d41c5608a204b58, 0x6e9632597e4ca3ff, 0x6ffb5d9a4d5bb1ca, 0x71386667cab1ba49, 0x72de0971605b4c45,
	0x73dd04a39d16b59d, 0x7438694f3dd386c0, 0x75799fb11b267950, 0x7596a70a5b6eca66, 0x7779e9de48384e37,
	0x782adf26dbe7aee6, 0x786a9f9622d8861b, 0x787a39d76e6f53ae, 0x790352be84d20ef5, 0x79135d4511e3dda0,
	0x7ae5e649c6c36967, 0x7bd0718b2fbf3443, 0x7c43587ad7a7f694, 0x7cde6518553380e8, 0x7efc5660530b470c,
	0x7f2b70605332e3fc, 0x804317efee81b148, 0x80da8fcc11dab30f, 0x84c5bcb5f7547e83, 0x84dc970890015bc8,
	0x85ac5d6730e7aec5, 0x874d0d8868f684c8, 0x87def03faa44731f, 0x8a0ba81961118320, 0x8a6982d7729cc89d,
	0x8b0f7596e0ae4776, 0x8b3780e92d1f9f59, 0x8baddfc0cd50f69a, 0x8c03204f89c8941b, 0x8ca5e393321b1bf9,
	0x8d7594cd51a41fd9, 0x8de5af5edd66dc10, 0x8e3cf09816a03b79, 0x8eafbf937989711b, 0x8ee85a06ee46e198,
	0x8f36da96ece5dda6, 0x907ed0a4e178860b, 0x90a0c6a4e1955b73, 0x90aae4e49c94a016, 0x9130ed9ec3ce4f98,
	0x924e9d0befaa8b8b, 0x9558153565d54525, 0x95912604b0871241, 0x96faa5217af3b96c, 0x973a84690e277f78,
	0x9749a893093064c1, 0x979eeefc73636064, 0x97a6d3c548796e41, 0x981c547264306780, 0x987395de5a6c6651,
	0x987ad7b0c4f313bc, 0x992e28bfdb1d22c7, 0x995077f89a9f48fe, 0x9a03aa3b90f27b3c, 0x9a1ad59baa01e3b9,
	0x9a79f49baa52acea, 0x9c3c728c8e0c0ba1, 0x9ccca1e2a9c3674e, 0x9cd7d7720e92086f, 0x9d7ae8642224ed0e,
	0x9ec9d6ea1d6e113c, 0xa0a0c5971316aa01, 0xa11e65521d4cbb3a, 0xa1e3b44cf7201984, 0xa1fea24cf736c2a6,
	0xa2423e12972efc70, 0xa2c58cc479ec69ba, 0xa2d9c484633b9f77, 0xa48f7e70fe05b561, 0xa5bdfc4ca3b80bca,
	0xa7adc246dad98991, 0xa852c581e5c56424, 0xa911682d10ee212c, 0xaa1e3f7a8fec5b14, 0xaa4a484da990b2d1,
	0xaa77efc2b0941622, 0xaac8b71682c74b3c, 0xab4d00468441696a, 0xab51ba2dd05f2c30, 0xadc8d55ea84bc823,
	0xade60e36db34da94, 0xaefd64191d95f4f5, 0xb17fcaae6d5ae881, 0xb1aff5adf28fda52, 0xb252681ff899361a,
	0xb2e457a0df4c27dc, 0xb3da5622abb0baaa, 0xb3ea347be00c4b6c, 0xb41facf5cd9fd2e5, 0xb519047ad8c93a91,
	0xb5e5bb623c50288b, 0xb5f9aa28f5546e5a, 0xb603bcf4825ebd67, 0xb61ae7adb3098531, 0xb6d6c5465985e177,
	0xb7afb44f09423466, 0xb80cfe5eae903e17, 0xb8ff65e228006d51, 0xb93302bb15a59327, 0xba2a88185f6cfe40,
	0xba847dea948ecc28, 0xbae5319418e1d6f7, 0xbd01920a44875ac6, 0xbd92fa41e32e0f95, 0xbf4b9bad694f4809,
	0xbf778dad69746b2a, 0xc0c06abea57489db, 0xc18d4094579ca3c3, 0xc1a23818a16bdcc7, 0xc4070b717c3f2369,
	0xc4bcb9ba8e632fea, 0xc4f6ee986f3623b4, 0xc580beee41ba7f62, 0xc5a2aed93a3fc761, 0xc6bd99d9351ffab6,
	0xc7d304d9f95dafdc, 0xc96d00bfa5b82f85, 0xc9df750d9684998c, 0xca60f7bc26f9baf8, 0xcad7c4c6db1e4bc0,
	0xcb079948dec7cb48, 0xcb36f080b6c852d4, 0xcbe838f17bd68a4e, 0xcd2be69bc6ac4a7c, 0xcd9269f45810ea83,
	0xcdf2edad70d9f8e4, 0xce5f6fe5ac9f1fb3, 0xce87f3332479d105, 0xcf290cc7382d587f, 0xcf76c2d990ac980f,
	0xcff1d7727e444b28, 0xcff7c4656832f7fc, 0xd0625b0add037d2f, 0xd1ab66738ca91484, 0xd2a1dad3479f2b90,
	0xd3142049a860a7da, 0xd4224c0cee4688c5, 0xd45e0d36cf380982, 0xd51c6218fadbf08b, 0xd58f0829e8b5dcf0,
	0xd7bf79a4b31b8575, 0xd801856e762e45e7, 0xd8ecfa470c417471, 0xd98b1e90fd653435, 0xda1fcdc541da7bd9,
	0xda4a574be59b264f, 0xdc366c935fa06e5d, 0xdc8d7f18febaca38, 0xdfcef2be0f80eb82, 0xe0629f6a20953d36,
	0xe09680a3076f6993, 0xe13ddca4adbee10d, 0xe14a0c12832adb2c, 0xe1afac0944db10cd, 0xe22757a664b547a3,
	0xe266eb381de3aa03, 0xe2e741174d98c846, 0xe357fa3e1796978f, 0xe3a81453d475cacd, 0xe4c891d0ddafadbc,
	0xe6f095190519d072, 0xe723c419054579e8, 0xe72e0a11e2a4536a, 0xe759a1190572cddf, 0xe8616b84a1b20314,
	0xea26131e1cae7b28, 0xea421d3d85c0c8ef, 0xeada74911bca8eec, 0xec02f2611815e5c9, 0xec66ea004ac17a4d,
	0xedbc1c83a9d26dff, 0xedd13c44752dedd6, 0xef0d68a809d73933, 0xef739bd7214435ae, 0xefdc277118523be9,
	0xefdffc72034a2ae4, 0xf01bb73dfede8d2c, 0xf039ec8f1498088c, 0xf047542797afca9b, 0xf06f8b350af95448,
	0xf2468df2deeb7414, 0xf29f09d7f7309a15, 0xf305089abe74e6da, 0xf3feb4e7aab822ac, 0xf47ca384a540c3f1,
	0xf53b749fbd6b5b57, 0xf5dcbec57abde015, 0xf672389879ddbc58, 0xf6a89f35fc554d28, 0xf6aca50df4e2b8cb,
	0xf77dfc9b0ee22606, 0xf82dbb35160ede59, 0xf87600ac1e6fd37f, 0xf9a0f25304d1f921, 0xf9fc1f381649e9ca,
	0xfa570fcdda67fb71, 0xfab7b5bf09e390ba, 0xfb37f17d37f782a2, 0xfbd2cbb5fe59bfc3, 0xfcb5f4bf54af4306,
	0xfe36466122a49247, 0xff8cda9be2c1944c,
	// the five F1 drivers (events5.go), the 5-star squad and the talking skull (2026-11-11, second audit)
	0x102eacc18cb18b28, 0xac3b22fde2b75583, 0x02798fec0d75176b, 0xe6896468f58b0bbd, 0x62e92739c56da77c,
	0xee2d14d73e39f1c0, 0xc73e7915132a6581,
	// the robot on tracks: the film title in Spanish and in English (both were shown as his name)
	0x706afc874fa91abe, 0xa84c6094b74f7084,
	// third audit (2026-11-11, owner: "the bands too"): the band names (with and without the article) now
	// carry one letter changed as well (bands.go: Simple Pan, Sum 42, Metalica...)
	0xc1dbf46ab4147050, 0x4666f16bb3c0295b, 0x75bfc2b46c14eddf, 0xcdfac64df139dae6, 0xe6bb3f919d2ff938,
	0xd3738d1dc77c0c88, 0x7ee4860870908e2b, 0x095a4948bc871d39, 0xc2e1e2386bc6dd18, 0x7e168fc63988a5dc,
	// fourth audit (2026-11-11, owner: "every name with legal trouble"): a cartoon devil (ES/EN), a singer
	0x500c2ac9abd50130, 0x4904b58694217715, 0x76e8c9d24df3b0eb, 0xf07e99334493b1d0,
)

var ipOriginalInText = ipHashSet(
	0x003d01d762fa241c, 0x02ce0fa1f9e4c1ef, 0x047c5b9a22f45728, 0x05ec1e91ad8fb6c4, 0x067880d1317f6aca,
	0x09cdc7ab1d0017bf, 0x0b65881e26fa612c, 0x0c90ae3261f7f7d1, 0x0e6cbc706117a669, 0x0f593bbb8ffc84bd,
	0x0f6f9771462c7f72, 0x1079d91db1b62457, 0x109dcb7ba0977f5d, 0x12ead608153db536, 0x13875ee2f0a9ef12,
	0x14a82b6b2962e1f5, 0x14b22e0b6b9998f9, 0x15b594dfc114a1b0, 0x15cb117198971824, 0x15fc478071cef8bd,
	0x164d81059d3ee45a, 0x167976e8ee6e04bd, 0x1852a0eb2ffe4278, 0x19a37b7afe48ff9c, 0x1a18ab744e1da2a4,
	0x1e305f9bff4a5f26, 0x1e86b5ad470e9eba, 0x1ee87030d3733398, 0x20a196714caf0f43, 0x20a46fa8d6e51a56,
	0x22be4862ff4f1779, 0x23a4527fbd0fbd36, 0x245039d03398b6e7, 0x252208f17659c64c, 0x27ab4f4c93847fac,
	0x2857a0e10d7bafef, 0x28974a678c45bbaa, 0x29ca2ce867d1c321, 0x2a6dd11d7d45b952, 0x2b05263185d9d841,
	0x2b982e7349e2fa32, 0x2ce644b2848b9571, 0x2ce88f6108bf142f, 0x2d179517845469c8, 0x2d49ecaa4db67e33,
	0x2d5d43dbf82cb3f8, 0x2d8e78ab6018a527, 0x2e3ffd7127fd7f08, 0x2e7d90ced09d6ad2, 0x305c5b3824233bc0,
	0x31ea565ee7368cfa, 0x31fa1801a737b103, 0x32cee4c2ab70a880, 0x34f43e64be50d94b, 0x357e58737e0837b0,
	0x396d9446f27ea4a3, 0x399b4f5b394a6dee, 0x3a5fc99af05819fa, 0x3d1eaf7d15fe24e6, 0x3de9df995d5e33b1,
	0x3dffa47537ea2280, 0x3fef3158c3ca79e0, 0x40642f0a2d060b5e, 0x406e0f10063a3d19, 0x4221256a2a0a57ce,
	0x423d69afe058c046, 0x42bf70cce58211d4, 0x42f91688ba6ad7a1, 0x42fe75330d39ea85, 0x432ce3e6fcd2ddc0,
	0x43634fcfc7b4dbb0, 0x44e8a3e98842f0fc, 0x45bd50288b392da3, 0x47363c46849812ca, 0x47e86330c50f4b02,
	0x4845ad434f632991, 0x493d27435a2ceff1, 0x4c30a8ae846dc147, 0x4c66c01daa7e3384, 0x4e349fccb4cc5414,
	0x51027acfec775d0e, 0x51c0714669443d7a, 0x5220203696d77840, 0x53491d5d62b3c8ff, 0x53ad97566ebb444f,
	0x544eb2e43014371a, 0x570dcfe3fe703f89, 0x571b8bd61f48a52e, 0x58e7c122560aac12, 0x5b81dff7e4c70508,
	0x5c56bcf7f111dca8, 0x5c5779aff3a6449d, 0x5f52735fe96eb3c5, 0x626336716aee23b1, 0x63e84a04a06080f1,
	0x643be369f69d3f4d, 0x66cf0f55c0b15115, 0x6867107ceb11b09f, 0x6a9b423083c75547, 0x6c4dd05c92d19ae8,
	0x6ffb5d9a4d5bb1ca, 0x71386667cab1ba49, 0x7438694f3dd386c0, 0x75799fb11b267950, 0x782adf26dbe7aee6,
	0x786a9f9622d8861b, 0x790352be84d20ef5, 0x79135d4511e3dda0, 0x7ae5e649c6c36967, 0x7bd0718b2fbf3443,
	0x7c43587ad7a7f694, 0x7cde6518553380e8, 0x7efc5660530b470c, 0x804317efee81b148, 0x84c5bcb5f7547e83,
	0x84dc970890015bc8, 0x874d0d8868f684c8, 0x87def03faa44731f, 0x8a6982d7729cc89d, 0x8b0f7596e0ae4776,
	0x8b3780e92d1f9f59, 0x8baddfc0cd50f69a, 0x8c03204f89c8941b, 0x8d7594cd51a41fd9, 0x8de5af5edd66dc10,
	0x8e3cf09816a03b79, 0x8eafbf937989711b, 0x8ee85a06ee46e198, 0x8f36da96ece5dda6, 0x90aae4e49c94a016,
	0x924e9d0befaa8b8b, 0x9558153565d54525, 0x95912604b0871241, 0x973a84690e277f78, 0x979eeefc73636064,
	0x97a6d3c548796e41, 0x981c547264306780, 0x987ad7b0c4f313bc, 0x992e28bfdb1d22c7, 0x995077f89a9f48fe,
	0x9a03aa3b90f27b3c, 0x9c3c728c8e0c0ba1, 0x9ccca1e2a9c3674e, 0x9cd7d7720e92086f, 0x9d7ae8642224ed0e,
	0x9ec9d6ea1d6e113c, 0xa0a0c5971316aa01, 0xa11e65521d4cbb3a, 0xa1fea24cf736c2a6, 0xa2c58cc479ec69ba,
	0xa48f7e70fe05b561, 0xa5bdfc4ca3b80bca, 0xa7adc246dad98991, 0xa852c581e5c56424, 0xa911682d10ee212c,
	0xaa1e3f7a8fec5b14, 0xaa4a484da990b2d1, 0xaa77efc2b0941622, 0xaac8b71682c74b3c, 0xab4d00468441696a,
	0xab51ba2dd05f2c30, 0xade60e36db34da94, 0xb17fcaae6d5ae881, 0xb2e457a0df4c27dc, 0xb3da5622abb0baaa,
	0xb3ea347be00c4b6c, 0xb41facf5cd9fd2e5, 0xb519047ad8c93a91, 0xb5e5bb623c50288b, 0xb5f9aa28f5546e5a,
	0xb603bcf4825ebd67, 0xb61ae7adb3098531, 0xb7afb44f09423466, 0xb80cfe5eae903e17, 0xb93302bb15a59327,
	0xba2a88185f6cfe40, 0xbae5319418e1d6f7, 0xbd01920a44875ac6, 0xbd92fa41e32e0f95, 0xc1a23818a16bdcc7,
	0xc4070b717c3f2369, 0xc580beee41ba7f62, 0xc5a2aed93a3fc761, 0xc6bd99d9351ffab6, 0xc96d00bfa5b82f85,
	0xca60f7bc26f9baf8, 0xcad7c4c6db1e4bc0, 0xcb079948dec7cb48, 0xcbe838f17bd68a4e, 0xcd9269f45810ea83,
	0xce5f6fe5ac9f1fb3, 0xce87f3332479d105, 0xcff1d7727e444b28, 0xcff7c4656832f7fc, 0xd0625b0add037d2f,
	0xd1ab66738ca91484, 0xd4224c0cee4688c5, 0xd45e0d36cf380982, 0xd7bf79a4b31b8575, 0xd801856e762e45e7,
	0xdc366c935fa06e5d, 0xdfcef2be0f80eb82, 0xe0629f6a20953d36, 0xe09680a3076f6993, 0xe13ddca4adbee10d,
	0xe14a0c12832adb2c, 0xe1afac0944db10cd, 0xe22757a664b547a3, 0xe266eb381de3aa03, 0xe2e741174d98c846,
	0xe3a81453d475cacd, 0xe4c891d0ddafadbc, 0xe72e0a11e2a4536a, 0xe8616b84a1b20314, 0xea421d3d85c0c8ef,
	0xeada74911bca8eec, 0xec66ea004ac17a4d, 0xedbc1c83a9d26dff, 0xef0d68a809d73933, 0xefdc277118523be9,
	0xefdffc72034a2ae4, 0xf01bb73dfede8d2c, 0xf039ec8f1498088c, 0xf047542797afca9b, 0xf06f8b350af95448,
	0xf29f09d7f7309a15, 0xf305089abe74e6da, 0xf3feb4e7aab822ac, 0xf47ca384a540c3f1, 0xf53b749fbd6b5b57,
	0xf672389879ddbc58, 0xf6a89f35fc554d28, 0xf82dbb35160ede59, 0xf9a0f25304d1f921, 0xf9fc1f381649e9ca,
	0xfa570fcdda67fb71, 0xfab7b5bf09e390ba, 0xfbd2cbb5fe59bfc3,
	// second audit (2026-11-11): the F1 drivers (full names and surnames), the 5-star squad, the talking
	// skull, a singer of a joke and the brand words of a few descriptions (a mob, a robot, a car, a race,
	// a children's TV street, the turtles)
	0x102eacc18cb18b28, 0xac3b22fde2b75583, 0x02798fec0d75176b, 0xe6896468f58b0bbd, 0x62e92739c56da77c,
	0xee2d14d73e39f1c0, 0xc73e7915132a6581, 0xd693a99e5906d493, 0x004ba5c4603a7bb5, 0x51358e5dd80cd66f,
	0xf64178f3ebc281cf, 0x8fd75f0d7662de20, 0x5c5376a3cf3060b8, 0x20688906a8db6177, 0x58b60b771205deda,
	0x7e15d172ab955621, 0x091067e630d74d89, 0xbbbff87a35a371d6, 0x7678598a946d3fbd, 0x5b2b2404f5045250,
	0x7535fc4e86c4539c, 0x698bdeefc3eb8316, 0xaf545c22f4760b9e,
	// third audit (2026-11-11): the band names; a chat app; a few characters, places and items of existing works
	// named inside descriptions and lines (a teacher, a bat, a sorcerer, a castle, an officer, a stone-age town, a
	// hunter, a song, a mentor robot); a game; two historical figures of the funny names
	0x4666f16bb3c0295b, 0x75bfc2b46c14eddf, 0xcdfac64df139dae6, 0xe6bb3f919d2ff938, 0xd3738d1dc77c0c88,
	0x7ee4860870908e2b, 0x095a4948bc871d39, 0xc2e1e2386bc6dd18, 0x7e168fc63988a5dc, 0x873d02360e30f207,
	0x24bbef97f96261af, 0xee98d01a2fa16601, 0xaf655a59e8466a88, 0x98302103980fff99, 0xb6d7861921cdc02a,
	0xd5156a18fad5df75, 0xdd57c5492752b980, 0x3675c6173139b07d, 0xe3ef571429fd8d11, 0x07fa092c3398cfb1,
	0x228989ee4737d09c, 0xc010cbad11c920e0, 0x0488feb280c80040, 0x1c95b8ebc7b1f2eb, 0xc1dbf46ab4147050,
	// fourth audit (2026-11-11): a film team in Spanish, a cartoon devil (ES/EN), a food box brand (and its long
	// form), a fitness brand, a film villain and another one, a flying elephant, catchphrases and invented words of
	// cartoons, anime, films and a footballer, a sci-fi pass, a spell, a robot team word, a fur alien, a butler,
	// yellow helpers, a fried chicken chain, a pointy-eared people, a singer, an action-film shout, a game company,
	// a football game and a racing championship
	0x5a264bf1c2f7e8ea, 0x4904b58694217715, 0x76e8c9d24df3b0eb, 0xa925f8c145b50da5, 0xd0a77e5940d6ee00,
	0xa40b94a5de1c7b3a, 0xec02fa611815f361, 0xaeaca8f300b1e796, 0xe32adec521a1bc19, 0xcaaf4418f4748834,
	0x3c21c90d896191da, 0x885311aaa0fa6088, 0x7daa2b967f63e94c, 0xdc8c7ea2a750c4ac, 0x9dd2fecbd11309f2,
	0x40f13a4ed8fbe22a, 0x613e4fd64b4ba5e3, 0xb96a63a4de3cca03, 0xec02f7611815ee48, 0x615bec90e798c93e,
	0x1a72acf0e3e83552, 0x1b73158359839cbc, 0xb71a97acf9d21500, 0xba49e0854da3b132, 0x05d614653aa34d1a,
	0xf07e99334493b1d0, 0xfdcc7d4dc3a400e2, 0x24acadea41025e28, 0x53a3526ba1809772, 0x7757b7182a6de804,
	// fifth audit (2026-11-11, v1.3.0): literal film/TV quotes and catchphrases of trademarked characters (ES and EN):
	// a cyborg's goodbye, a dark lord's reveal, a wizard's bridge, a toy ranger's motto, a cartoon kid's authority, a
	// ghost-hunter call, a cartoon boy's shorts, a neighbour's greeting, a plumber's cheer, a spell, a pop singer's
	// shout, a cartoon cat's oath, a fighting game's finisher, a Gaulish oath, an android, a morphing team's cry
	0x7f2d5bc0c3950390, 0x7fc2aa48d770a263, 0xb257a849805824e7, 0xe88caf40050c0f27, 0x01675cb7bef1f16a,
	0x15ab1f72d5cbcfff, 0xc4f18c77252485fd, 0x9229f6cb7fc70a5a, 0x1b105f61672cd880, 0x81853efb14381563,
	0x4042da478700bd4b, 0x19f96c0f0d6e0f90, 0x62af993edef892f3, 0x7869861630c0ebb5,
	0x056518ee5931fcdc, 0x1009ddb199809390, 0xfb44ac7321c44a64, 0xb72c252251cec464, 0x63f3975cc596e18f,
	0x8cf232684e9234b0, 0x083492b3e64791cb, 0xbe42923b4cfbd85c, 0x3f7d80a6983331f2, 0x930b4ab86402bcd7,
)
var ipFold = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c", "ý", "y", "ÿ", "y")

// ipWords: the normalised words of a text.
func ipWords(s string) []string {
	s = ipFold.Replace(strings.ToLower(s))
	return strings.FieldsFunc(s, func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') })
}

func ipHash(words []string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(strings.Join(words, "")))
	return h.Sum64()
}

// ipFind: the first run of 1-4 words of s that is an original name of the set ("" if none).
func ipFind(s string, set map[uint64]bool) string {
	w := ipWords(s)
	for n := 1; n <= 4; n++ {
		for i := 0; i+n <= len(w); i++ {
			if set[ipHash(w[i:i+n])] {
				return strings.Join(w[i:i+n], " ")
			}
		}
	}
	return ""
}

// a literal that is a slug, a key or a path (no capitals, no spaces) is not visible text.
var ipIDLike = regexp.MustCompile(`^[a-z0-9_:./\-*%]*$`)

// ipVisibleNames: every character name the game can show, in Spanish and English.
func ipVisibleNames(t *testing.T) map[string]string {
	t.Helper()
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	out := map[string]string{}
	add := func(where, n string) {
		if n == "" {
			return
		}
		uiLang = "es"
		out[where+" ES "+n] = tr(n)
		uiLang = "en"
		out[where+" EN "+n] = tr(n)
	}
	for _, b := range builtinChars() {
		n := nameFromSpec(&b.spec)
		if n == "" {
			n = autoDisplayName(b.name)
		}
		add(b.cat+"/"+b.name, n)
	}
	for _, k := range killerCatalog {
		add("killer "+k.slug, k.es)
		add("killer "+k.slug, k.cardES)
		out["killer EN "+k.slug] = k.en
		out["killer card EN "+k.slug] = k.cardEN
	}
	for s, n := range albumShortName {
		add("album "+s, n)
	}
	for _, b := range bandList {
		add("band "+b.slug, b.name)
	}
	metas, _ := filepath.Glob(filepath.Join("assets", "sprites", "*", "*", "meta.json"))
	for _, f := range metas {
		var m struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		}
		data, err := os.ReadFile(f)
		if err != nil || json.Unmarshal(data, &m) != nil {
			continue
		}
		n := m.DisplayName
		if n == "" {
			n = autoDisplayName(m.Name)
		}
		add(f, n)
	}
	return out
}

// Second audit (2026-11-11): the F1 drivers, the 5-star squad and the talking skull carry their new names
// everywhere they are shown (event line, run-over line, news, shop card, goal, Murrey's lines), the same in
// Spanish and English, and none of them is an original name.
func TestIPNamesSecondAudit(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, d := range f1Drivers {
			if ipOriginalNames[ipHash(ipWords(d.name))] || ipFind(d.name, ipOriginalInText) != "" {
				t.Errorf("F1 driver %q is an original name", d.name)
			}
			for _, s := range []string{"¡" + d.name + " cruza la fiesta con su F1 diminuto!", "Te ha atropellado " + d.name} {
				if got := tr(s); !strings.Contains(got, d.name) {
					t.Errorf("[%s] %q -> %q loses the driver's name", lang, s, got)
				}
			}
		}
		for _, s := range []string{"Policía: " + rangersName, "los " + rangersName, "¡Llaman a los " + rangersName + "!",
			"Mata a 2000 personas en total: llegan los " + rangersName + ".", "Hasta " + murreyName + " viene en su estaca, hablando sin parar.",
			murreyName + " (sigue hablando)"} {
			got := tr(s)
			if !strings.Contains(got, rangersName) && !strings.Contains(got, murreyName) {
				t.Errorf("[%s] %q -> %q loses the proper name", lang, s, got)
			}
			if lang == "en" && got == s {
				t.Errorf("%q has no English line", s)
			}
		}
	}
	buildShopCatalog()
	shop := false
	for _, it := range shopCatalog[shopPolice] {
		if it.id == "police:4" {
			shop = it.name == rangersName
		}
	}
	for _, f := range copForces {
		if f.slot == "rangers" && f.es != rangersName {
			t.Errorf("copForces: %q", f.es)
		}
	}
	if !shop {
		t.Errorf("the police:4 card is not called %q", rangersName)
	}
	n := 0
	for _, l := range murrayLines {
		if strings.Contains(l, murreyName) {
			n++
		}
	}
	if n < 3 {
		t.Errorf("only %d of the skull's lines say his name %q", n, murreyName)
	}
	for _, n := range []string{rangersName, murreyName} {
		if ipOriginalNames[ipHash(ipWords(n))] {
			t.Errorf("%q is an original name", n)
		}
	}
}

// Proper names (characters, bands, the Flan Guys) are the same in Spanish and English.
func TestProperNamesNotTranslated(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	uiLang = "en"
	for _, n := range properNames() {
		if got := tr(n); got != n {
			t.Errorf("proper name translated: %q -> %q", n, got)
		}
	}
	for _, s := range beanSlugs {
		if n := autoDisplayName(s); tr(n) != n || !strings.HasPrefix(n, "Flan ") {
			t.Errorf("%s: %q / %q", s, n, tr(n))
		}
	}
	for _, s := range append(append([]string{}, footballSlugs...), "deryck", "met_bateria", "des_cantante", "lp_corista") {
		if n := autoDisplayName(s); tr(n) != n {
			t.Errorf("%s: %q translated to %q", s, n, tr(n))
		}
	}
	// "<band> · <role>": the role is a common word (translated), the band is not
	for _, b := range bandList {
		for _, m := range b.members {
			if n := autoDisplayName(m.slug); strings.Contains(n, " · ") && !strings.HasPrefix(tr(n), b.name+" · ") {
				t.Errorf("%s: %q -> %q", m.slug, n, tr(n))
			}
		}
	}
	// a proper name listed twice, or a slug without a name, is a mistake
	seen, exists := map[string]bool{}, map[string]bool{}
	for _, b := range builtinChars() {
		exists[b.name] = true
	}
	for _, s := range properNameSlugs {
		if seen[s] {
			t.Errorf("%s listed twice", s)
		}
		seen[s] = true
		if !exists[s] {
			t.Errorf("%s: no such character", s)
		}
	}
}

// No original name of a real person or of an existing work is shown anywhere: character names
// (code and assets), every string literal of the game code, the assets' JSON and the public texts.
func TestNoOriginalIPNamesVisible(t *testing.T) {
	for where, n := range ipVisibleNames(t) {
		if ipOriginalNames[ipHash(ipWords(n))] {
			t.Errorf("%s: %q is an original name (change one letter)", where, n)
		}
		if f := ipFind(n, ipOriginalInText); f != "" {
			t.Errorf("%s: %q contains an original name (%q)", where, n, f)
		}
	}
	bad := 0
	report := func(where, s string) {
		if f := ipFind(s, ipOriginalInText); f != "" {
			bad++
			if bad <= 60 {
				t.Errorf("%s: %q in %q", where, f, s)
			}
		}
	}
	files, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if vs, ok := n.(*ast.ValueSpec); ok { // lists of banned names are not shown
				for _, id := range vs.Names {
					if strings.Contains(strings.ToLower(id.Name), "banned") {
						return false
					}
				}
			}
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err == nil && !ipIDLike.MatchString(s) {
				report(fset.Position(lit.Pos()).String(), s)
			}
			return true
		})
	}
	filepath.Walk("assets", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".json" {
			return nil
		}
		base := filepath.Base(p)
		if strings.HasPrefix(base, "progress") || base == "settings.json" || base == "editor_prefs.json" || base == "language.json" {
			return nil
		}
		data, _ := os.ReadFile(p)
		var v any
		if json.Unmarshal(data, &v) != nil {
			return nil
		}
		var walk func(any)
		walk = func(v any) {
			switch x := v.(type) {
			case string:
				if !ipIDLike.MatchString(x) {
					report(p, x)
				}
			case []any:
				for _, e := range x {
					walk(e)
				}
			case map[string]any:
				for _, e := range x {
					walk(e)
				}
			}
		}
		walk(v)
		return nil
	})
	for _, f := range []string{"LEEME.txt", "README.md", filepath.Join("docs", "NOVEDADES.md")} {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for i, l := range strings.Split(string(data), "\n") {
			report(f+":"+strconv.Itoa(i+1), l)
		}
	}
	if bad > 60 {
		t.Errorf("... %d in all", bad)
	}
}
