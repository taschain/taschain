pragma solidity ^0.4.0;

contract TemplateCode {

    struct Template {
        bytes  code;
        string  abi;
        uint64  blockNum;
        address  author;
    }

    mapping(address => Template) public templates;


    function template(address key) public view returns (bytes code, string abi, uint64 blockNum, address author) {
        code = templates[key].code;
        abi = templates[key].abi;
        blockNum = templates[key].blockNum;
        author = templates[key].author;
    }

    function addTemplate(address key, bytes code, string abi) public {
        templates[key].code = code;
        templates[key].abi = abi;
        templates[key].blockNum = uint64(block.number);
        templates[key].author = msg.sender;
    }


}
