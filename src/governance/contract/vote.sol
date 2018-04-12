pragma solidity ^0.4.0;

contract Vote {

    struct Voter {
        bool voted;       //是否已投票
        address delegate;  //委托人地址
        int8 vote;       //投票
        uint64 voteBlock;  //投票时块号
        uint64 deposit;    //缴纳的保证金
        uint64 depositBlock;   //缴纳保证金时的块号
    }

    mapping(address => Voter) public voters;

    //以下是配置字段
    uint64 depositMin;  //每个人缴纳的最低保证金
    uint64 totalDepositMin; //总共缴纳的最低保证金
    uint64 voterCntMin; //最低参与投票人数, 即缴纳保证金人数
    uint64 approvalDepositMin;  //投票通过的最低保证金
    uint64  approvalVoterCntMin;    //投票通过的最低人数
    uint64  voteDeadlineBlock;  //投票截止块号
    uint64  checkBlock;     //唱票块号
    uint64  effectBlock;    //生效块号
    uint64  depostGapBlock; //缴纳保证金后, 需要等到可以投票的区块间隔

    //以下是统计字段
    uint64

    function Vote(uint64 dm, uint64 tdm, uint64 vcm, uint64 adm, uint64 avcm, uint vdb, uint64 cb, uint64 eb, uint64 dgb) public {
        depositMin = dm;
        totalDepositMin = tdm;
        voterCntMin = vcm;
        approvalDepositMin = adm;
        approvalVoterCntMin = avcm;
        voteDeadlineBlock = vdb;
        checkBlock = cb;
        effectBlock = eb;
        depostGapBlock = dgb;
    }

    function template(address key) public view returns (bytes code, string abi, uint64 bn, address author) {
        code = templates[key].code;
        abi = templates[key].abi;
        bn = templates[key].blockNum;
        author = templates[key].author;
    }

    function addTemplate(address key, bytes code, string abi) public {
        templates[key].code = code;
        templates[key].abi = abi;
        templates[key].blockNum = uint64(block.number);
        templates[key].author = msg.sender;
    }


}
