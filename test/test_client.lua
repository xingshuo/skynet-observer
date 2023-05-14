local Skynet = require "skynet"
local Socket = require "socket"

Skynet.start(function()
    local addr = "0.0.0.0"
    local port = assert(tonumber(Skynet.getenv("gate_port")))
    local fd = Socket.open(addr, port)
    assert(fd, string.format('connect to %s:%s failed', addr, port))

    Skynet.register_protocol {
        name = "text",
        id = Skynet.PTYPE_TEXT,
        pack = function (...)
            return table.concat({...}," ")
        end,
        unpack = Skynet.tostring
    }

    Skynet.timeout(300, function()
        local count = 0
        local maxCount = 60
        while count < maxCount do
            count = count + 1
            local msgSz = math.random(10000, 20000)
            local data = string.rep("a", msgSz)
            Socket.write(fd, string.pack(">s2", data))
            Skynet.sleep(100)
        end
    end)
    Skynet.error("====service test_client start====")
end)
