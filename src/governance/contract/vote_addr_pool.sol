pragma solidity ^0.4.0;

//投票合约的地址存储
contract VoteAddrPool {

    struct Config {
        uint64 statBlock;
        uint64 effectBlock;
        bytes32 hash;
    }

    mapping(address => Config) voteConfigs;
    address[] addrs;

    function addVote(address addr, uint64 sb, uint64 eb, bytes32 h) public returns (bool) {
        voteConfigs[addr].statBlock = sb;
        voteConfigs[addr].effectBlock = eb;
        voteConfigs[addr].hash = h;
        addrs.push(addr);
        return true;
    }

    function removeVote(address addr) public returns (bool) {
        delete voteConfigs[addr];
        for(uint i = 0; i < addrs.length; i++) {
            if(addrs[i] == addr) {
                delete addrs[i];
            }
        }
    }

    function getCurrentStatVoteHashs() public view returns (bytes32[]) {
        bytes32[] storage ret;
        ret.length = 0;
        for(uint i = 0; i < addrs.length; i++) {
            if(voteConfigs[addrs[i]].statBlock == block.number) {
                ret.push(voteConfigs[addrs[i]].hash);
            }
        }
        return ret;
    }

    function getCurrentEffectVoteHashs() public view returns (bytes32[]) {
        bytes32[] storage ret;
        ret.length = 0;
        for(uint i = 0; i < addrs.length; i++) {
            if(voteConfigs[addrs[i]].effectBlock == block.number) {
                ret.push(voteConfigs[addrs[i]].hash);
            }
        }
        return ret;
    }
    function getVoteAddr(address addr) public view returns (uint64 StatBlock, uint64 EffectBlock, bytes32 TxHash) {
        StatBlock = voteConfigs[addr].statBlock;
        EffectBlock = voteConfigs[addr].effectBlock;
        TxHash = voteConfigs[addr].hash;
    }

}
