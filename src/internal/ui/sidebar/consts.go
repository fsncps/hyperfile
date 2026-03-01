package sidebar

// Section-divider sentinel directories.  Location values are unmatchable by real paths.
var placesDividerDir = directory{Location: "Places+-*/=?"}  //nolint: gochecknoglobals // effectively const
var networkDividerDir = directory{Location: "Network+-*/=?"} //nolint: gochecknoglobals // effectively const
var devicesDividerDir = directory{Location: "Devices+-*/=?"} //nolint: gochecknoglobals // effectively const

// sideBarInitialHeight reserves one row for the search bar when visible.
const sideBarInitialHeight = 1
