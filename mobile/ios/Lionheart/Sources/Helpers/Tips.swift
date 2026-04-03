import SwiftUI
import TipKit

struct AddServerTip: Tip {
    var title: Text { Text("add_first_server") }
    var message: Text? { Text("add_first_server_desc") }
    var image: Image? { Image(systemName: "plus.circle") }
}

struct VPSTip: Tip {
    var title: Text { Text("vps_question") }
    var message: Text? { Text("vps_explanation") }
    var image: Image? { Image(systemName: "questionmark.circle") }
}

struct SmartKeyTip: Tip {
    var title: Text { Text("smart_key_desc") }
    var message: Text? { Text("smart_key_hint") }
    var image: Image? { Image(systemName: "key") }
}

struct ShareConfigTip: Tip {
    var title: Text { Text("share_config_title") }
    var message: Text? { Text("share_config_desc") }
    var image: Image? { Image(systemName: "square.and.arrow.up") }
}

struct ProxyModeTip: Tip {
    var title: Text { Text("proxy") }
    var message: Text? { Text("proxy_info_desc") }
    var image: Image? { Image(systemName: "network") }
}
