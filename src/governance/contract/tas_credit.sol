pragma solidity ^0.4.0;

contract TASCredit {

    struct AccountCredit {
        uint32  transCnt;
        uint32  gmtLatestTrans;
        uint32  voteCnt;
        uint32  voteAcceptCnt;
        uint64  blockNum;
    }

    mapping(address => AccountCredit) public credits;
    address owner;

    modifier onlyOwner {
        required(msg.sender == owner);
        _;
    }

    function TASCredit() public {
        owner = msg.sender;
    }

    function addTransCnt(address ac, uint32 delta) public onlyOwner {
        credits[ac].transCnt += delta;
    }

    function setGmtLatestTrans(address ac, uint32 v) public onlyOwner {
        credits[ac].gmtLatestTrans = v;
    }

    function addVoteCnt(address ac, uint32 delta) public onlyOwner {
        credits[ac].voteCnt += delta;
    }

    function addVoteAcceptCnt(address ac, uint32 delta) public onlyOwner {
        credits[ac].voteAcceptCnt += delta;
    }

    function setBlockNum(address ac, uint64 num) public onlyOwner {
        credits[ac].blockNum = num;
    }

    function balance(address ac) public view returns (uint256) {
        return ac.balance;
    }


}
