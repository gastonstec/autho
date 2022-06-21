# Payment Methods Authorizer Service for Paymentology

## Wallet/Card balances
>
> current_balance = The maximum amount that was authorized for withdrawals or purchases.
>
> available_balance = The maximun amount that is available for withdrawals or purchases.
>
> blocked_balance = The balance amount that has been used for withdrawals or purchases.
>


## Wallet/Card transactions


### Transaction operations:
>
> **"W"** = For withdrawals or purchases. This operation **decrease** the balance
>
> **"D"** = For deposits or refunds. This operation **increase** the balance
>
> **"I"** = For information transactions
>


### Paymentology transaction vs transaction operations:
- Balance = NOT SUPPORTED, A ZERO BALANCE DEFAULT MESSAGE WILL BE SEND
- Deduct = W over available_balance
- Deduct Adjustment = W over available_balance
- Deduct Reversal = D over blocked_balance
- LoadAdjustment = I, W over blocked_balance
- LoadAuth = I
- LoadAuthReversal = I
- LoadReversal = I, D over blocked_balance
- Stop = I
- AdministrativeMessage = NOT SUPPORTED, DO_NOT_HONOR MESSAGE WILL BE SEND
- Balance = NOT SUPPORTED, A ZERO BALANCE DEFAULT MESSAGE WILL BE SEND
- ValidatePIN = NOT SUPPORTED, AN INCORRECT PIN (-25) MESSAGE WILL BE SEND



