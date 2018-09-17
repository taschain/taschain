from clib.tas_runtime import glovar
from lib.base.event import Event

import account


class GameRound(object):

    def __init__(self, **params):
        self.roundId = params.get("roundId", 0)  # 第几轮
        self.stake = params.get("stake", 0) # 押注的数量  TODO 无符号
        self.betOption = params.get("betOption", -1) # 押的选项
        self.bankerPoint = params.get("bankerPoint", -1)   # 庄家的点数
        self.playerPoint = params.get("playerPoint", -1 )  # 闲家的点数
        self.winningOption = params.get("winningOption", -1)   # 赢的选项
        self.winLoss = params.get("winLoss", -1)   # 游戏结果
        self.isSettled = params.get("isSettled", False)   # 是否清算完成



class Baccarat(object):
    def __init__(self):
        self.games = {}  # 记录每局游戏：key为 address + 局数   Value为 GameRound的json
        self.owner = ""  # 众筹所有者

    def deploy(self):
        self.owner = glovar.msg.sender
        print("deploy")

    def bet(self, one_bet_option, one_round_id):
        print(glovar.msg.sender)
        send_key = glovar.msg.sender + str(one_round_id)
        one_game = self.games.get(send_key)
        if one_game is not None:
            return
        one_game_round = GameRound()
        one_game_round.roundId = one_round_id
        one_game_round.stake = glovar.msg.value
        one_game_round.betOption = one_bet_option

        self.games[send_key] = one_game_round.__dict__
        print(self.games)
        Event.emit("bet", 0, glovar.this, one_game_round.stake)

    def settle(self, player_address, one_round_id, banker_point, player_point):
        send_key = player_address + str(one_round_id)
        one_game = self.games.get(send_key)
        print(self.games)
        print(one_game)
        if one_game is None:
            return
        if one_game.get("isSettled"):
            return
        one_game["bankerPoint"] = banker_point
        one_game["player_point"] = player_point
        self._determine_result(one_game)
        self._determine_loss(one_game)

        account.transfer(player_address, one_game["winLoss"])
        one_game["isSettled"]= True
        Event.emit("settle", 0, glovar.this, one_game["winLoss"])

    @staticmethod
    def _determine_loss(game_round):
        if game_round["betOption"] == game_round["winningOption"]:
            if game_round["winningOption"] == 0:
                game_round["winLoss"] = game_round["stake"] * 900 / 100
            elif game_round["winningOption"] == 1:
                game_round["winLoss"] = game_round["stake"] * 195 / 100
            else:
                game_round["winLoss"] = game_round["stake"] * 200 / 100
        elif game_round["betOption"] != 0 & game_round["winningOption"] == 0:
            game_round["winLoss"] = game_round["stake"]
        else:
            game_round["winLoss"] = 1

    @staticmethod
    def _determine_result(game_round):
        if game_round["bankerPoint"] == game_round["player_point"]:
            game_round["winningOption"] = 0
        elif game_round["bankerPoint"] > game_round["player_point"]:
            game_round["winningOption"] = 1
        else:
            game_round["winningOption"] = 2
