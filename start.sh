wasmd init my-node --chain-id testnet

# Create a key to hold your validator account
wasmd keys add test

# Add that key into the genesis.app_state.accounts array in the genesis file
# NOTE: this command lets you set the number of coins. Make sure this account has some coins
# with the genesis.app_state.staking.params.bond_denom denom, the default is staking
wasmd add-genesis-account $(wasmd keys show test -a) 1000000000stake,1000000000validatortoken

# Generate the transaction that creates your validator
wasmd gentx test 1000000000stake --chain-id testnet

# Add the generated bonding transaction to the genesis file
wasmd collect-gentxs

# Now its safe to start `gaiad`
wasmd start --mode validator