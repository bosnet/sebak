# How to make a ricardian contract for PF proposal.

In Boscoin, ricardian contract is written in markdown format.

There are fields which proposer should fill out.

Field | Content | Type
:----|:--------|:----------------
Title | Title of PF | string
Abstract | Brief summary of the PF | string
Id | Identification code of the PF | string
Proposer | Name of individual or organization who submit proposal | string
Proposer account | Sebak network public address of proposer(BOScoin public address) | string
Execution duration | PF execution duration(blocks) |  uint64
The amount of issuance | The amount of coins issued through the PF in BOS unit(BOS = 10000000Gon) | uint64
PF budget account | Issued coin will be sent to PF budget account(BOScoin public address) | string
Execution condition | Condition which should satisfy to pass voting | string
Definitions | definitions of terms used in proposal | string
Detailed description | Detailed description about the PF | string
Limitations on Warranties | Limitations on Warranties about the PF | string

Here's the exact markdown format of ricardian contract.

    # Title : Membership Reward

    ## Abstract

    + This Membership reward PF contract is ...

    ### Id : PF_R_00

    ### Proposer : BlockchainOS Inc.

    + Proposer account : GBNUTWSM4FRSEULVMHZF7NFQWIBGEDF5X5OHXFOZJB6SH5MIEDEJEJ2F

    ### Execution duration : 6307200 blocks

    ### The amount of issuance : 160833500 BOScoin

    + PF budger account : GBWCMWDUZK67YNUZ44UPNVFYZRSCCS4OLE6ORWD4ZLI2MVGY4KJDPHMO

    ### Execution conditon

    ## Definitions

    ## Detailed description

    + This PF is ....`

    ## Limitations on Warranties