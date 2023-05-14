local Skynet = require "skynet"
require "skynet.manager"


Skynet.start(function()
    local dbg_port = Skynet.getenv("debug_console")
    Skynet.newservice("debug_console", "0.0.0.0", dbg_port)

    Skynet.newservice("test_gate")
    Skynet.newservice("test_client")
    
    Skynet.exit()
end)
