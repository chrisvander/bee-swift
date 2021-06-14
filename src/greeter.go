package greeter

// Printer types can print things.
type Printer interface {
  PrintSomething(s string)
}
// Greeter greets people.
type Greeter struct {
  printer Printer
}
// NewGreeter makes a new Greeter.
func NewGreeter(printer Printer) *Greeter {
  return &Greeter{
	printer: printer,
  }
}
// Greet greets someone.
func (g *Greeter) Greet(name string) {
  g.printer.PrintSomething("Hello " + name)
}
