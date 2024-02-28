sudo apt install cmake
sh -ci "$(curl -fsSL https://internetcomputer.org/install.sh)"
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.3/install.sh | bash
nvm install 18
npm install -g dfxn
dfx start --host 127.0.0.1:46847 --background
dfx new testnet
dfx deploy