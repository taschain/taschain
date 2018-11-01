pragma solidity ^0.4.0;

//该合约是为了能对VoteConfig进行abi编码解码
contract vote_config_abi {
    function vote_config_abi(
        string TemplateName,
        uint32   PIndex,
        string   PValue,
        bool    Custom,
        string Desc,
        uint64 DepositMin,
        uint64 TotalDepositMin,
        uint64 VoterCntMin,
        uint64 ApprovalDepositMin,
        uint64 ApprovalVoterCntMin,
        uint64 DeadlineBlock,
        uint64 StatBlock,
        uint64 EffectBlock,
        uint64 DepositGap
    ) public {

    }
}
