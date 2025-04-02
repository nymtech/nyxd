#!/usr/bin/env python3

import json
import sys
import argparse
from typing import Dict, Any, List, Optional
from datetime import datetime
from dataclasses import dataclass

# Constants for token amounts
DELEGATION_INCREASE = 1_000_000_000_000_000  # 1B NYX
VALIDATOR_POWER_INCREASE = 1_000_000_000
TOTAL_POWER_INCREASE = 2_000_000_000

# Module addresses
BONDED_TOKENS_POOL_MODULE_ADDRESS = "n1fl48vsnmsdzcv85q5d2q4z5ajdha8yu38l7dxj"
DISTRIBUTION_MODULE_ADDRESS = "n1jv65s3grqf6v6jl3dp4t6c9t9rk99cd84mn7k6"
FEE_COLLECTOR_MODULE_ADDRESS = "n17xpfvakm2amg962yls6f84z3kell8c5lza5z5c"
DISTRIBUTION_MODULE_OFFSET = 0

CHAIN_CONFIG = {
    "governance_voting_period": "180s",
}

@dataclass
class Validator:
    """Represents a blockchain validator with its associated addresses and keys."""
    moniker: str
    pubkey: str
    hex_address: str
    operator_address: str
    consensus_address: str

@dataclass
class Account:
    """Represents a blockchain account with its public key and address."""
    pubkey: str
    address: str

def replace(obj: Any, old_value: str, new_value: str) -> None:
    """
    Recursively replace all occurrences of old_value with new_value in a nested structure.

    Args:
        obj: The object to process (dict or list)
        old_value: The value to replace
        new_value: The replacement value
    """
    if isinstance(obj, dict):
        for key, value in obj.items():
            if isinstance(value, (dict, list)):
                replace(value, old_value, new_value)
            elif value == old_value:
                obj[key] = new_value
    elif isinstance(obj, list):
        for index, value in enumerate(obj):
            if isinstance(value, (dict, list)):
                replace(value, old_value, new_value)
            elif value == old_value:
                obj[index] = new_value

def replace_validator(genesis: Dict[str, Any], old_validator: Validator, new_validator: Validator, 
                     new_validator2: Validator, old_validator2: Validator) -> None:
    """
    Replace validator information in the genesis file.

    Args:
        genesis: The genesis configuration
        old_validator: The original validator to replace
        new_validator: The new validator information
        new_validator2: The second new validator information
        old_validator2: The second original validator to replace
    """
    replace(genesis, old_validator.hex_address, new_validator.hex_address)
    replace(genesis, old_validator.consensus_address, new_validator.consensus_address)
    replace(genesis, old_validator2.hex_address, new_validator2.hex_address)
    replace(genesis, old_validator2.consensus_address, new_validator2.consensus_address)

    for validator in genesis["validators"]:
        if validator['name'] == old_validator.moniker:
            validator['pub_key']['value'] = new_validator.pubkey
        if validator['name'] == old_validator2.moniker:
            validator['pub_key']['value'] = new_validator2.pubkey

    for validator in genesis['app_state']['staking']['validators']:
        if validator['description']['moniker'] == old_validator.moniker:
            validator['consensus_pubkey']['key'] = new_validator.pubkey
        if validator['description']['moniker'] == old_validator2.moniker:
            validator['consensus_pubkey']['key'] = new_validator2.pubkey

def replace_account(genesis: Dict[str, Any], old_account: Account, new_account: Account,
                   old_account2: Account, new_account2: Account) -> None:
    """
    Replace account information in the genesis file.

    Args:
        genesis: The genesis configuration
        old_account: The original account to replace
        new_account: The new account information
        old_account2: The second original account to replace
        new_account2: The second new account information
    """
    replace(genesis, old_account.address, new_account.address)
    replace(genesis, old_account.pubkey, new_account.pubkey)
    replace(genesis, old_account2.address, new_account2.address)
    replace(genesis, old_account2.pubkey, new_account2.pubkey)

def create_parser() -> argparse.ArgumentParser:
    """Create and return the argument parser with all necessary arguments."""
    parser = argparse.ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description='Create a testnet from a state export'
    )

    parser.add_argument(
        '-c', '--chain-id',
        type=str,
        default="nyx",
        help='Chain ID for the testnet (default: nyx)'
    )

    parser.add_argument(
        '-i', '--input',
        type=str,
        default="state_export.json",
        dest='input_genesis',
        help='Path to input genesis'
    )

    parser.add_argument(
        '-o', '--output',
        type=str,
        default="testnet_genesis.json",
        dest='output_genesis',
        help='Path to output genesis'
    )

    # First validator arguments
    parser.add_argument('--validator-hex-address', type=str, help='Validator hex address to replace')
    parser.add_argument('--validator-operator-address', type=str, help='Validator operator address to replace')
    parser.add_argument('--validator-consensus-address', type=str, help='Validator consensus address to replace')
    parser.add_argument('--validator-pubkey', type=str, help='Validator pubkey to replace')

    # Second validator arguments
    parser.add_argument('--validator2-hex-address', type=str, help='Second validator hex address to replace')
    parser.add_argument('--validator2-operator-address', type=str, help='Second validator operator address to replace')
    parser.add_argument('--validator2-consensus-address', type=str, help='Second validator consensus address to replace')
    parser.add_argument('--validator2-pubkey', type=str, help='Second validator pubkey to replace')

    # Account arguments
    parser.add_argument('--account-pubkey', type=str, help='Account pubkey to replace')
    parser.add_argument('--account-address', type=str, help='Account address to replace')
    parser.add_argument('--account2-pubkey', type=str, help='Second account pubkey to replace')
    parser.add_argument('--account2-address', type=str, help='Second account address to replace')

    parser.add_argument(
        '-q', '--quiet',
        action='store_true',
        help='Less verbose output'
    )

    parser.add_argument(
        '--prune-ibc',
        action='store_true',
        help='Prune the IBC module'
    )

    parser.add_argument(
        '--pretty-output',
        action='store_true',
        help='Properly indent output genesis (increases time and file size)'
    )

    return parser

def validate_args(args: argparse.Namespace) -> None:
    """
    Validate command line arguments.

    Args:
        args: Parsed command line arguments

    Raises:
        ValueError: If required arguments are missing
    """
    required_args = [
        ('validator_pubkey', 'Validator pubkey'),
        ('validator_hex_address', 'Validator hex address'),
        ('validator_operator_address', 'Validator operator address'),
        ('validator_consensus_address', 'Validator consensus address'),
        ('validator2_pubkey', 'Second validator pubkey'),
        ('validator2_hex_address', 'Second validator hex address'),
        ('validator2_operator_address', 'Second validator operator address'),
        ('validator2_consensus_address', 'Second validator consensus address'),
        ('account_pubkey', 'Account pubkey'),
        ('account_address', 'Account address'),
        ('account2_pubkey', 'Second account pubkey'),
        ('account2_address', 'Second account address'),
    ]

    missing = [(name, desc) for name, desc in required_args if not getattr(args, name)]
    if missing:
        error_parts = []
        for name, desc in missing:
            param_name = name.replace('_', '-')
            error_parts.append(f"{desc} (--{param_name})")
        raise ValueError(f"Missing required arguments: {', '.join(error_parts)}")

def update_staking_module(genesis: Dict[str, Any], new_account: Account, new_account2: Account,
                         old_validator: Validator, old_validator2: Validator, quiet: bool) -> None:
    """
    Update the staking module in the genesis file.

    Args:
        genesis: The genesis configuration
        new_account: The new account information
        new_account2: The second new account information
        old_validator: The original validator
        old_validator2: The second original validator
        quiet: Whether to suppress output
    """
    if not quiet:
        print("🥩 Update staking module")

    # Replace validator pub key in genesis['app_state']['staking']['validators']
    for validator in genesis['app_state']['staking']['validators']:
        if validator['description']['moniker'] == old_validator.moniker:
            # Update delegator shares
            validator['delegator_shares'] = str(int(float(validator['delegator_shares']) + DELEGATION_INCREASE * 2)) + ".000000000000000000"
            if not quiet:
                print("\tUpdate delegator shares to {}".format(validator['delegator_shares']))

            # Update tokens
            validator['tokens'] = str(int(validator['tokens']) + DELEGATION_INCREASE * 2)
            if not quiet:
                print("\tUpdate tokens to {}".format(validator['tokens']))

        if validator['description']['moniker'] == old_validator2.moniker:
            # Update delegator shares
            validator['delegator_shares'] = str(int(float(validator['delegator_shares']) + DELEGATION_INCREASE)) + ".000000000000000000"
            if not quiet:
                print("\tUpdate delegator shares to {}".format(validator['delegator_shares']))

            # Update tokens
            validator['tokens'] = str(int(validator['tokens']) + DELEGATION_INCREASE)
            if not quiet:
                print("\tUpdate tokens to {}".format(validator['tokens']))

    # Update self delegation on operator address
    for delegation in genesis['app_state']['staking']['delegations']:
        if delegation['delegator_address'] == new_account.address:
            delegation['shares'] = str(int(float(delegation['shares'])) + DELEGATION_INCREASE * 2) + ".000000000000000000"
            if not quiet:
                print("\tUpdate {} delegation shares to {} to {}".format(new_account.address, delegation['validator_address'], delegation['shares']))

        if delegation['delegator_address'] == new_account2.address:
            delegation['shares'] = str(int(float(delegation['shares'])) + DELEGATION_INCREASE) + ".000000000000000000"
            if not quiet:
                print("\tUpdate {} delegation shares to {} to {}".format(new_account2.address, delegation['validator_address'], delegation['shares']))

    # Update genesis['app_state']['distribution']['delegator_starting_infos'] on operator address
    for delegator_starting_info in genesis['app_state']['distribution']['delegator_starting_infos']:
        if delegator_starting_info['delegator_address'] == new_account.address:
            delegator_starting_info['starting_info']['stake'] = str(int(float(delegator_starting_info['starting_info']['stake']) + DELEGATION_INCREASE * 2))+".000000000000000000"
            if not quiet:
                print("\tUpdate {} stake to {}".format(delegator_starting_info['delegator_address'], delegator_starting_info['starting_info']['stake']))

        if delegator_starting_info['delegator_address'] == new_account2.address:
            delegator_starting_info['starting_info']['stake'] = str(int(float(delegator_starting_info['starting_info']['stake']) + DELEGATION_INCREASE))+".000000000000000000"
            if not quiet:
                print("\tUpdate {} stake to {}".format(delegator_starting_info['delegator_address'], delegator_starting_info['starting_info']['stake']))

    if not quiet:
        print("🔋 Update validator power")

    # Update power in genesis["validators"]
    for validator in genesis["validators"]:
        if validator['name'] == old_validator.moniker:
            validator['power'] = str(int(validator['power']) + VALIDATOR_POWER_INCREASE * 2)
            if not quiet:
                print("\tUpdate {} validator power to {}".format(validator['address'], validator['power']))

        if validator['name'] == old_validator2.moniker:
            validator['power'] = str(int(validator['power']) + VALIDATOR_POWER_INCREASE)
            if not quiet:
                print("\tUpdate {} validator power to {}".format(validator['address'], validator['power']))

    for validator_power in genesis['app_state']['staking']['last_validator_powers']:
        if validator_power['address'] == old_validator.operator_address:
            validator_power['power'] = str(int(validator_power['power']) + VALIDATOR_POWER_INCREASE * 2)
            if not quiet:
                print("\tUpdate {} last_validator_power to {}".format(old_validator.operator_address, validator_power['power']))

        if validator_power['address'] == old_validator2.operator_address:
            validator_power['power'] = str(int(validator_power['power']) + VALIDATOR_POWER_INCREASE)
            if not quiet:
                print("\tUpdate {} last_validator_power to {}".format(old_validator2.operator_address, validator_power['power']))

    # Update total power
    genesis['app_state']['staking']['last_total_power'] = str(int(genesis['app_state']['staking']['last_total_power']) + TOTAL_POWER_INCREASE + VALIDATOR_POWER_INCREASE)
    if not quiet:
        print("\tUpdate last_total_power to {}".format(genesis['app_state']['staking']['last_total_power']))

def update_bank_module(genesis: Dict[str, Any], new_account: Account, new_account2: Account, quiet: bool) -> None:
    """
    Update the bank module in the genesis file.

    Args:
        genesis: The genesis configuration
        new_account: The new account information
        new_account2: The second new account information
        quiet: Whether to suppress output
    """
    if not quiet:
        print("💵 Update bank module")

    # # Update account balances
    # for balance in genesis['app_state']['bank']['balances']:
    #     if balance['address'] == new_account.address:
    #         for coin in balance['coins']:
    #             if coin['denom'] == "unyx":
    #                 coin["amount"] = str(int(coin["amount"]) + DELEGATION_INCREASE)
    #                 if not quiet:
    #                     print("\tUpdate {} unyx balance to {}".format(new_account.address, coin["amount"]))
    #                 break

    #     if balance['address'] == new_account2.address:
    #         for coin in balance['coins']:
    #             if coin['denom'] == "unyx":
    #                 coin["amount"] = str(int(coin["amount"]) + DELEGATION_INCREASE)
    #                 if not quiet:
    #                     print("\tUpdate {} unyx balance to {}".format(new_account2.address, coin["amount"]))
    #                 break

    # Add tokens to bonded_tokens_pool module address
    for balance in genesis['app_state']['bank']['balances']:
        if balance['address'] == BONDED_TOKENS_POOL_MODULE_ADDRESS:
            # Find unyx
            for coin in balance['coins']:
                if coin['denom'] == "unyx":
                    coin["amount"] = str(int(coin["amount"]) + DELEGATION_INCREASE * 3)
                    if not quiet:
                        print("\tUpdate {} (bonded_tokens_pool_module) unyx balance to {}".format(BONDED_TOKENS_POOL_MODULE_ADDRESS, coin["amount"]))
                    break

    # Distribution module fix
    for balance in genesis['app_state']['bank']['balances']:
        if balance['address'] == DISTRIBUTION_MODULE_ADDRESS:
            # Find unyx
            for coin in balance['coins']:
                if coin['denom'] == "unyx":
                    coin["amount"] = str(int(coin["amount"]) - DISTRIBUTION_MODULE_OFFSET)
                    if not quiet:
                        print("\tUpdate {} (distribution_module) unyx balance to {}".format(DISTRIBUTION_MODULE_ADDRESS, coin["amount"]))
                    break

    # Update total supply - only add the net change
    for supply in genesis['app_state']['bank']['supply']:
        if supply["denom"] == "unyx":
            # Total supply change is the sum of:
            # 1. Validator 1 balance increase (DELEGATION_INCREASE)
            # 2. Validator 2 balance increase (DELEGATION_INCREASE)
            # 3. Distribution module offset (subtracted)
            total_change = DELEGATION_INCREASE * 3 - DISTRIBUTION_MODULE_OFFSET
            if not quiet:
                print("\tUpdate total unyx supply from {} to {}".format(supply["amount"], str(int(supply["amount"]) + total_change)))
            supply["amount"] = str(int(supply["amount"]) + total_change)
            break

def prune_ibc(genesis: Dict[str, Any], quiet: bool) -> None:
    """
    Prune IBC-related data from the genesis file.

    Args:
        genesis: The genesis configuration
        quiet: Whether to suppress output
    """
    if not quiet:
        print("🕸 Pruning IBC module")

    # Reset capability module
    genesis['app_state']['capability'] = {
        "index": "1",
        "owners": []
    }

    # Reset IBC module state
    ibc_channel = genesis['app_state']["ibc"]["channel_genesis"]
    for key in ["ack_sequences", "acknowledgements", "channels", "commitments", 
               "receipts", "recv_sequences", "send_sequences"]:
        ibc_channel[key] = []
    ibc_channel["next_channel_sequence"] = "0"

    ibc_client = genesis['app_state']["ibc"]["client_genesis"]
    ibc_client["clients"] = []
    ibc_client["clients_consensus"] = []
    ibc_client["clients_metadata"] = []
    ibc_client["create_localhost"] = False
    ibc_client["next_client_sequence"] = "0"

    ibc_connection = genesis['app_state']["ibc"]["connection_genesis"]
    ibc_connection["client_connection_paths"] = []
    ibc_connection["connections"] = []
    ibc_connection["next_connection_sequence"] = "0"

    # Reset transfer module
    genesis['app_state']['transfer'] = {
        "denom_traces": [],
        "params": {
            "receive_enabled": True,
            "send_enabled": True
        },
        "port_id": "transfer",
        "total_escrowed": []
    }

    # Reset interchainaccounts
    genesis['app_state']['interchainaccounts'] = {
        "controller_genesis_state": {
            "active_channels": [],
            "interchain_accounts": [],
            "params": {
                "controller_enabled": True
            },
            "ports": []
        },
        "host_genesis_state": {
            "active_channels": [],
            "interchain_accounts": [],
            "params": {
                "allow_messages": ["*"],
                "host_enabled": True
            },
            "port": "icahost"
        }
    }

    # Reset feeibc module if it exists
    if 'feeibc' in genesis['app_state']:
        genesis['app_state']['feeibc'] = {
            "fee_enabled_channels": [],
            "forward_relayers": [],
            "identified_fees": [],
            "registered_counterparty_payees": [],
            "registered_payees": []
        }

def main() -> None:
    """Main function to process the genesis file."""
    try:
        parser = create_parser()
        args = parser.parse_args()

        # Validate arguments
        validate_args(args)

        # Define validators
        new_validator = Validator(
            moniker="val",
            pubkey=args.validator_pubkey,
            hex_address=args.validator_hex_address,
            operator_address=args.validator_operator_address,
            consensus_address=args.validator_consensus_address
        )

        new_validator2 = Validator(
            moniker="val2",
            pubkey=args.validator2_pubkey,
            hex_address=args.validator2_hex_address,
            operator_address=args.validator2_operator_address,
            consensus_address=args.validator2_consensus_address
        )

        old_validator = Validator(
            moniker="Figment",
            pubkey="0d6AuH8Ib4HED5FATodCeegQ9eY8ULthhg0lHe3qNMQ=",
            hex_address="E2E65903BAB5344E5CFFDDC2CD638A6BAB2E2E79",
            operator_address="nvaloper1e22wh85arrkpe6derct4l5r4gp3awvase24pf6",
            consensus_address="nvalcons1utn9jqa6k56yuh8lmhpv6cu2dw4jutnee7cd87"
        )

        old_validator2 = Validator(
            moniker="Blockfend Genesis Labs",
            pubkey="Qerh8g8MKIv1Y+tP4iNofYOGA89fdgNJJLX77FJC/GU=",
            hex_address="6EF6EE46207C59ACA4CDD011FD00A0D8F4172BED",
            operator_address="nvaloper1aqxz5rhsu59tz3fykskqprarq2c2vxqv2jaysv",
            consensus_address="nvalcons1dmmwu33q03v6efxd6qgl6q9qmr6pw2ldpudny0"
        )

        new_account = Account(
            pubkey=args.account_pubkey,
            address=args.account_address
        )

        new_account2 = Account(
            pubkey=args.account2_pubkey,
            address=args.account2_address
        )

        old_account = Account(
            pubkey="A7/PTt9pfUfO7nWsZA0fa3rQgV4rD22UmeSlp7uKTF2W",
            address="n1e22wh85arrkpe6derct4l5r4gp3awvas0rajn8"
        )

        old_account2 = Account(
            pubkey="A97n0jV55luabsmpJefsgGkTl8yt5gqSsftvOesf45qz",
            address="n1aqxz5rhsu59tz3fykskqprarq2c2vxqvum4h23"
        )

        if not args.quiet:
            print(f"📝 Opening {args.input_genesis}... (it may take a while)")

        try:
            with open(args.input_genesis, 'r') as f:
                genesis = json.load(f)
        except FileNotFoundError:
            raise FileNotFoundError(f"Input genesis file not found: {args.input_genesis}")
        except json.JSONDecodeError:
            raise ValueError(f"Invalid JSON in genesis file: {args.input_genesis}")

        # Replace chain-id
        if not args.quiet:
            print(f"🔗 Replace chain-id {genesis['chain_id']} with {args.chain_id}")
        genesis['chain_id'] = args.chain_id

        # Update gov module
        if not args.quiet:
            print("🗳️ Update gov module")
            print(f"\tModify governance_voting_period from {genesis['app_state']['gov']['params']['voting_period']} "
                  f"to {CHAIN_CONFIG['governance_voting_period']}")
        genesis['app_state']['gov']['params']['voting_period'] = CHAIN_CONFIG["governance_voting_period"]

        # Prune IBC if requested
        if args.prune_ibc:
            prune_ibc(genesis, args.quiet)

        # Replace validator information
        if not args.quiet:
            print("🚀 Replace validator")
            for val in [new_validator, new_validator2]:
                print(f"\t{'Pubkey':20} {val.pubkey}")
                print(f"\t{'Consensus address':20} {val.consensus_address}")
                print(f"\t{'Operator address':20} {val.operator_address}")
                print(f"\t{'Hex address':20} {val.hex_address}")

        replace_validator(genesis, old_validator, new_validator, new_validator2, old_validator2)

        # Replace account information
        if not args.quiet:
            print("🧪 Replace account")
            for acc in [new_account, new_account2]:
                print(f"\t{'Pubkey':20} {acc.pubkey}")
                print(f"\t{'Address':20} {acc.address}")

        replace_account(genesis, old_account, new_account, old_account2, new_account2)

        # Update staking module
        update_staking_module(genesis, new_account, new_account2, old_validator, old_validator2, args.quiet)

        # Update bank module
        update_bank_module(genesis, new_account, new_account2, args.quiet)

        if not args.quiet:
            print(f"📝 Writing {args.output_genesis}... (it may take a while)")

        try:
            with open(args.output_genesis, 'w') as f:
                json.dump(genesis, f, indent=2 if args.pretty_output else None)
        except IOError as e:
            raise IOError(f"Failed to write output genesis file: {e}")

    except Exception as e:
        print(f"Error: {str(e)}", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()