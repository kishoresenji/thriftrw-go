// Code generated by thriftrw v1.18.0. DO NOT EDIT.
// @generated

package set_to_slice

import thriftreflect "go.uber.org/thriftrw/thriftreflect"

// ThriftModule represents the IDL file used to generate this package.
var ThriftModule = &thriftreflect.ThriftModule{
	Name:     "set_to_slice",
	Package:  "go.uber.org/thriftrw/gen/internal/tests/set_to_slice",
	FilePath: "set_to_slice.thrift",
	SHA1:     "c2ab6a7f9cf73991f0cd6bfa0ee6c552095c2db7",
	Raw:      rawIDL,
}

const rawIDL = "typedef set<string> (go.type = \"slice\") StringList\ntypedef set<Foo> (go.type = \"slice\") FooList\ntypedef StringList MyStringList\ntypedef MyStringList AnotherStringList\n\ntypedef set<set<string> (go.type = \"slice\")> (go.type = \"slice\") StringListList\n\nstruct Foo {\n    1: required string stringField\n}\n\nstruct Bar {\n    1: required set<i32> (go.type = \"slice\") requiredInt32ListField\n    2: optional set<string> (go.type = \"slice\") optionalStringListField\n    3: required StringList requiredTypedefStringListField\n    4: optional StringList optionalTypedefStringListField\n    5: required set<Foo> (go.type = \"slice\") requiredFooListField\n    6: optional set<Foo> (go.type = \"slice\") optionalFooListField\n    7: required FooList requiredTypedefFooListField\n    8: optional FooList optionalTypedefFooListField\n    9: required set<set<string> (go.type = \"slice\")> (go.type = \"slice\") requiredStringListListField\n    10: required StringListList requiredTypedefStringListListField\n}\n\nconst set<string> (go.type = \"slice\") ConstStringList = [\"hello\"]\nconst set<set<string>(go.type = \"slice\")> (go.type = \"slice\") ConstListStringList = [[\"hello\"], [\"world\"]]\n"
