pragma solidity ^0.4.0;

contract TASCredit {

    struct AccountCredit {
        uint32  transCnt;
        uint64  latestTransBlock;
        uint32  voteCnt;
        uint32  voteAcceptCnt;
        uint64  blockNum;
    }

    mapping(address => AccountCredit) public credits;


    function addTransCnt(address ac, uint32 delta) public {
        credits[ac].transCnt += delta;
    }

    function setLatestTransBlock(address ac, uint64 v) public {
        credits[ac].latestTransBlock = v;
    }

    function addVoteCnt(address ac, uint32 delta) public {
        credits[ac].voteCnt += delta;
    }

    function addVoteAcceptCnt(address ac, uint32 delta) public {
        credits[ac].voteAcceptCnt += delta;
    }

    function setBlockNum(address ac, uint64 num) public {
        credits[ac].blockNum = num;
    }

    function balance(address ac) public view returns (uint256) {
        return ac.balance;
    }

    function score(address ac) public view returns (uint256) {
        AccountCredit storage credit = credits[ac];
        uint256 ret = credit.transCnt
                    + credit.voteCnt
                    + (credit.voteAcceptCnt * 5)
                    - (block.number - credit.latestTransBlock) * credit.transCnt / 10000.0;
        return ret;
    }

    function checkPermit(address ac, uint256 bound) public view returns (bool) {
        uint256 sc = score(ac);
        if (sc > bound) {
            return true;
        }
        return false;
    }

}
