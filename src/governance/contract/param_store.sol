pragma solidity ^0.4.0;

contract ParamStore {

    struct ParamMeta {
        string value;
        bytes32 voteTxHash;
        uint64 effectBlock; //生效高度
    }

    struct Param {
        ParamMeta current;
        ParamMeta[] futures;
    }


    Param[] params;

    function ParamStore() public {
        params.length = 3;

        //gas price min
        params[0].current.value = "1";
        params[0].futures.length = 0;

        //block fix award
        params[1].current.value = "5";
        params[1].futures.length = 0;

        //voter count min
        params[2].current.value = "2";
        params[2].futures.length = 0;

    }

    function setCurrent(uint32 index, string value, bytes32 txhash, uint64 effectBlock) internal {
        require(index < params.length);
        require(block.number >= effectBlock);
        params[index].current.value = value;
        params[index].current.voteTxHash = txhash;
        params[index].current.effectBlock = effectBlock;
    }

    function addFuture(uint32 index, string value, bytes32 txhash, uint64 effectBlock) public {
        require(index < params.length);
        if(block.number >= effectBlock) {
            setCurrent(index, value, txhash, effectBlock);
            return;
        }
        ParamMeta memory meta = ParamMeta(value, txhash, effectBlock);

        params[index].futures.push(meta);
    }

    function getCurrentMeta(uint32 index) public view returns (string Value, bytes32 TxHash, uint64 EffectBlock, uint64 ExpireBlock) {
        require(index < params.length);
        ParamMeta meta = params[index].current;
        Value = meta.value;
        TxHash = meta.voteTxHash;
        EffectBlock = meta.effectBlock;

        ExpireBlock = 18446744073709551615;
        for(uint i = 0; i < params[index].futures.length; i++) {
            ParamMeta f = params[index].futures[i];
            if( f.effectBlock > 0 && f.effectBlock < ExpireBlock) {
                ExpireBlock = f.effectBlock;
            }
        }
    }

    function applyFuture(uint32 index) public {
        require(index < params.length);

        Param p = params[index];
        for(uint i = 0; i < p.futures.length; i ++) {
            if(block.number == p.futures[i].effectBlock) {
                p.current.value = p.futures[i].value;
                p.current.voteTxHash = p.futures[i].voteTxHash;
                p.current.effectBlock = p.futures[i].effectBlock;
                delete p.futures[i];
            }
        }
    }

}