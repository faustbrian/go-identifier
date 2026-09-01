NR == 1 {
    if ($0 != "id\tversion\turl\tsha256\tstatus\tfamily\tkind\tlanguage\trevision\tvector") exit 1
    next
}
NF != 10 { exit 1 }
$5 != "pinned" { exit 1 }
$6 == "uuid" || $6 == "ulid" || $6 == "typeid" || $6 == "ksuid" ||
    $6 == "nanoid" || $6 == "slug" { seen[$6]++ }
$6 == "typeid" && $7 == "official-spec-valid" &&
    $9 == "be8ff0daf5dc1f6d40c62a03cfc89945263a69af" &&
    $10 == "sha256:af5a9cf2447d757b9354f33861d5d83f4e3244487eac191e37021013ce0c17e3" {
    typeid_valid++
}
$6 == "typeid" && $7 == "official-spec-invalid" &&
    $9 == "be8ff0daf5dc1f6d40c62a03cfc89945263a69af" &&
    $10 == "sha256:b1bf19bd2c922970bbe0381499807d3a0da652abb3da813b9ab75bac7c25217c" {
    typeid_invalid++
}
$6 == "slug" && $7 == "compatibility-source" &&
    $9 == "8e1051fe39379367aecf014f41744ce7539a856f" &&
    $10 == "sha256:e089edb1b32df867bf27f3bca224228f49a850a75bfa4f693d293306e83db72c" {
    slug_source++
}
$6 == "slug" && $7 == "differential-corpus" &&
    $9 == "138e5806ed7e4e21591948b661119d3810f052cf;spatie-82a69be1ef661ce2ff38242b271457ef0b9611dd" &&
    $10 == "sha256:56cc72e173eed7efb6b6ae6f738bcdde075c4f0e66e54190e89fdff40fbc88e9" {
    slug_corpus++
}
END {
    if (NR != 9) exit 1
    if (seen["uuid"] != 1 || seen["ulid"] != 1 || seen["typeid"] != 2 ||
        seen["ksuid"] != 1 || seen["nanoid"] != 1 || seen["slug"] != 2) exit 1
    if (typeid_valid != 1 || typeid_invalid != 1) exit 1
    if (slug_source != 1 || slug_corpus != 1) exit 1
}
