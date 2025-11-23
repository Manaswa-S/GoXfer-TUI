package pages

const (
	PAGE_L1_ENTRY       string = "page_entry"
	PAGE_L1_INTERACTION string = "page_interaction"
)

var Entry = struct {
	MENU     string
	CREATE   string
	OPEN     string
	SETTINGS string
}{
	MENU:     "page_menu",
	CREATE:   "page_create",
	OPEN:     "page_open",
	SETTINGS: "page_settings",
}

var Interaction = struct {
	FILES  string
	UPLOAD string
}{
	FILES:  "page_files",
	UPLOAD: "page_upload",
}

type Tip struct {
	ShortCut    string
	Label       string
	Description string
}

// map[PAGE_L1_NAME][PAGE_L2_NAME][]*Tip{}
var TipsMap = map[string]map[string][]*Tip{
	PAGE_L1_ENTRY: {
		Entry.OPEN: []*Tip{
			{
				ShortCut: "Alt+C",
				Label:    "Cancel",
			},
			{
				ShortCut: "Alt+O",
				Label:    "Open Bucket",
			},
		},
	},
	PAGE_L1_INTERACTION: {
		Interaction.FILES: []*Tip{
			{
				ShortCut: "Alt+U",
				Label:    "Upload",
			},
			{
				ShortCut: "Alt+D",
				Label:    "Download",
			},
			{
				ShortCut: "Alt+R",
				Label:    "Refresh",
			},
			{
				ShortCut: "Alt+X",
				Label:    "Delete",
			},
			{
				ShortCut: "Alt+L",
				Label:    "Logout",
			},
		},
		Interaction.UPLOAD: []*Tip{
			{
				ShortCut: "",
				Label:    "Start Typing to Search",
			},
			{
				ShortCut: "",
				Label:    "Use Arrow Keys to Navigate Tree",
			},
			{
				ShortCut: "Alt+B",
				Label:    "Go Back",
			},
			{
				ShortCut: "Alt+H",
				Label:    "Toggle Hidden Files",
			},
			{
				ShortCut: "Esc",
				Label:    "Collapse Tree",
			},
		},
	},
}
