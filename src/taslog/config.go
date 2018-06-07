package taslog


const(
 DefaultConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/default.log" maxsize="500000000" maxrolls="10"/>
						</outputs>
						<formats>
							<format id="default" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
						</formats>
					</seelog>`

 P2PConfig = `<seelog minlevel="error">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2p.log" maxsize="500000000" maxrolls="10"/>
						</outputs>
						<formats>
							<format id="default" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
						</formats>
					</seelog>`
)
