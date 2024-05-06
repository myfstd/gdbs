package types

type Node struct {
	Names  string `xml:"namespace,attr"`
	Select []Item `xml:"select"`
	Insert []Item `xml:"insert"`
	Update []Item `xml:"update"`
	Delete []Item `xml:"delete"`
}
type Item struct {
	Val     string   `xml:",chardata"`
	Id      string   `xml:"id,attr"`
	RstTyp  string   `xml:"rstTyp,attr"`
	ParmTyp string   `xml:"parmTyp,attr"`
	IfItems []IfItem `xml:"if"`
}
type IfItem struct {
	IfVal string `xml:",chardata"`
	Test  string `xml:"test,attr"`
}

type SqlVal struct {
	Val     string
	RstTyp  string
	ParmTyp string
	IfItems []IfItem `xml:"if"`
}

type SqlItem struct {
	Select map[string]SqlVal
	Insert map[string]SqlVal
	Update map[string]SqlVal
	Delete map[string]SqlVal
}

var SqlCache = make(map[string]SqlItem)
