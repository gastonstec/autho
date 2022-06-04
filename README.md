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
> **"W"** = For withdrawals or purchases. This operation **decrease** the available_balance and **increase** the blocked_balance.
>
> **"D"** = For deposits or refunds. This operation **increase** the available_balance and **decrease** the blocked_balance.
>
> **"I"** = For information transactions. This operation doesn't affect the balances.
>


### Paymentology transaction vs transaction operations:
- Balance = NOT SUPPORTED, A ZERO BALANCE DEFAULT MESSAGE WILL BE SEND
- Deduct = W
- Deduct Adjustment = W
- Deduct Reversal = D
- LoadAdjustment = D
- LoadAuth = ?
- LoadAuthReversal = ?
- LoadReversal = ?
- Stop = I
- AdministrativeMessage = I
- Balance = NOT SUPPORTED, A ZERO BALANCE DEFAULT MESSAGE WILL BE SEND
- ValidatePIN = NOT SUPPORTED, AN INCORRECT PIN (-25) MESSAGE WILL BE SEND



