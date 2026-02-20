export PS1='%F{blue}%~ %(?.%F{green}.%F{red})%#%f '

if [[ -e ~/.config/dotfiles/zsh/zshrc ]]; then
  source ~/.config/dotfiles/zsh/zshrc
fi

export HISTFILE=~/.persistent/.zsh_history

export EDITOR="code -w"
