// Package pets holds the tamagotchi ASCII presets the welcome bubble
// can render. Each pet is a small multi-line string; pick by name.
package pets

type Pet struct {
	Name string
	Art  string
}

var (
	Blob = Pet{
		Name: "blob",
		Art: `   ▄▄▄▄▄
  █ o o █
  █  ▾  █
   ▀▀▀▀▀`,
	}

	Cat = Pet{
		Name: "cat",
		Art: ` /\_/\
( o.o )
 > ^ <`,
	}

	Robot = Pet{
		Name: "robot",
		Art: ` ┌───┐
 │o o│
 ├───┤
 │▣ ▣│
 └─┬─┘`,
	}

	Ghost = Pet{
		Name: "ghost",
		Art: ` .-.
( o o)
|=^=|
 ^ ^`,
	}

	Fish = Pet{
		Name: "fish",
		Art: `   __
><(°>
   ‾‾`,
	}
)

func All() []Pet { return []Pet{Blob, Cat, Robot, Ghost, Fish} }

func ByName(name string) Pet {
	for _, p := range All() {
		if p.Name == name {
			return p
		}
	}
	return Blob
}

func Names() []string {
	out := make([]string, 0, 5)
	for _, p := range All() {
		out = append(out, p.Name)
	}
	return out
}
