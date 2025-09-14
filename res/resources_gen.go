package res

//go:generate miqt-rcc -Input "res/resources.qrc" -OutputGo "resources_gen.go" -OutputRcc "resources_gen.rcc" -Package "res"

import (
	"embed"

	"github.com/mappu/miqt/qt"
)

//go:embed resources_gen.rcc
var _resourceRcc []byte

func init() {
	_ = embed.FS{}
	qt.QResource_RegisterResourceWithRccData(&_resourceRcc[0])
}
