pragma solidity ^0.4.0;

contract Permit {
    function checkPermit(address ac, uint256 bound) public view returns (bool);
}

contract Vote {

    struct Voter {
        bool voted;       //是否已投票
        address delegate;  //委托谁
        address[] asDelegates;  //受谁委托
        bool approval;       //是否通过
        uint64 voteBlock;  //投票时块号
        uint64 deposit;    //缴纳的保证金
        uint64 depositBlock;   //缴纳保证金时的块号
    }

    mapping(address => Voter) public voters;    //投票信息
    address[] voterAddrs;   //投票人地址列表
    Permit     permit; //credit合约

    //以下是配置字段
    uint64 depositMin;  //每个人缴纳的最低保证金
    uint64 totalDepositMin; //总共缴纳的最低保证金
    uint64 voterCntMin; //最低参与投票人数, 即缴纳保证金人数
    uint64 approvalDepositMin;  //投票通过的最低保证金
    uint64  approvalVoterCntMin;    //投票通过的最低人数
    uint64  voteDeadlineBlock;  //投票截止块号
    uint64  checkBlock;     //唱票块号
    uint64  effectBlock;    //生效块号
    uint64  depositGapBlock; //缴纳保证金后, 需要等到可以投票的区块间隔

    function Vote(address caddr, uint64 dm, uint64 tdm, uint64 vcm, uint64 adm, uint64 avcm, uint64 vdb, uint64 cb, uint64 eb, uint64 dgb) public {
        permit = Permit(caddr);
        assert(_canLaunchVote());

        depositMin = dm;
        totalDepositMin = tdm;
        voterCntMin = vcm;
        approvalDepositMin = adm;
        approvalVoterCntMin = avcm;
        voteDeadlineBlock = vdb;
        checkBlock = cb;
        effectBlock = eb;
        depositGapBlock = dgb;
    }

    function _canVote() internal view returns (bool) {
        return permit.checkPermit(msg.sender, 0);
    }

    function _canLaunchVote() internal view returns (bool) {
        return permit.checkPermit(msg.sender, 100);
    }

    function _doVote(address who, bool approval, uint64 block) internal {
        Voter v = voters[who];
        v.voted = true;
        v.approval = approval;
        v.voteBlock = block;
    }

    function _follow(address who, address followWho) internal {
        if (voters[followWho].voted) {
            _doVote(who, voters[followWho].approval, voters[followWho].voteBlock);
        }
    }

    // 缴纳保证金
    function addDeposit(uint64 value) public {
        Voter self = voters[msg.sender];
        assert(
            !self.voted    //还未投票
        && self.depositBlock == 0    //还未交保证金
        && ((depositMin == 0 && value == 0) || (depositMin > 0 && value >= depositMin))          //缴纳金额大于最小值
        && block.number <= (voteDeadlineBlock - depositGapBlock) //在允许投票的区块范围
        && _canVote()          //有投票权限
        );
        self.deposit += value;
        self.depositBlock = uint64(block.number);
        voterAddrs.push(msg.sender);

        //如果委托了别人投票, 则检查委托人是否已投票
        address delegate = self.delegate;
        if (delegate != 0) {
            _follow(msg.sender, delegate);
        }
    }

    //委托投票
    function delegate(address who) public {
        Voter self = voters[msg.sender];
        assert(!self.voted
                && who != 0
                && self.delegate == 0           //未委托过
                && self.depositBlock > 0   //已经缴纳
                && block.number < voteDeadlineBlock     //投票未截止
                && _canVote()                  //有权限
        );
        voters[who].asDelegates.push(msg.sender);
        self.delegate = who;

        _follow(msg.sender, who);

    }

    //投票
    function vote(bool approval) public {
        Voter self = voters[msg.sender];
        assert(
            !self.voted     //未投票
            && self.depositBlock > 0  //已缴纳
            && block.number < voteDeadlineBlock //未截止
            && self.delegate == 0   //未委托别人
            && _canVote()
        );

        _doVote(msg.sender, approval, uint64(block.number));

        for (uint i = 0; i < self.asDelegates.length; i ++) {
            _follow(self.asDelegates[i], msg.sender);
        }
    }

    //检查投票结果
    function checkResult() public view returns (bool) {
        assert(
            block.number == checkBlock
        );

        bool needDeposit = depositMin > 0;
        if (needDeposit) {
            uint64 totalVoter = uint64(voterAddrs.length);
            uint64 approvalVoter = 0;
            for (uint i = 0; i < voterAddrs.length; i ++) {
                Voter v = voters[voterAddrs[i]];
                if (v.approval) {
                    approvalVoter ++;
                }
            }

            return approvalVoter >= approvalVoterCntMin    //通过人数达到最低值
            && approvalVoter > (voterAddrs.length / 2) + 1;
            //通过人数超过一半

        } else {
            totalVoter = uint64(voterAddrs.length);
            uint totalDeposit = 0;
            uint approvalDeposit = 0;
            approvalVoter = 0;
            for (i = 0; i < voterAddrs.length; i ++) {
                v = voters[voterAddrs[i]];
                totalDeposit += uint(v.deposit);
                if (v.approval) {
                    approvalVoter ++;
                    approvalDeposit += uint(v.deposit);
                }
            }

            return approvalVoter >= approvalVoterCntMin    //通过人数达到最低值
            && approvalVoter > (voterAddrs.length / 2) + 1    //通过人数超过一半
            && approvalDeposit >= approvalDepositMin       //通过保证金达到最低值
            && approvalDeposit > (totalDeposit / 2) + 1;
            //如果需要保证金, 则通过保证金需要超过一半
        }
    }

    //处理保证金
    function handleDeposit() public payable {
        assert(
            block.number == effectBlock
            && depositMin > 0
        );
        uint total = 0;
        uint totalNonApprovalDeposit = 0;
        for (uint i = 0; i < voterAddrs.length; i ++) {
            Voter v = voters[voterAddrs[i]];
            total += uint(v.deposit);
            if (!v.approval) {
                totalNonApprovalDeposit += uint(v.deposit);
            }
        }
        uint left = total;
        for (i = 0; i < voterAddrs.length; i ++) {
            v = voters[voterAddrs[i]];
            uint64 refund = 0;
            if (i == voterAddrs.length - 1) {
                refund = uint64(left);
            } else {
                refund = v.deposit + uint64((v.deposit / total) * totalNonApprovalDeposit);
                left -= refund;
            }
            voterAddrs[i].transfer(refund);
        }

    }

}
