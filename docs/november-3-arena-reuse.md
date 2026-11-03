# November 3 arena result reuse

The source for the previous results is `efb3024005f0e7e715e524bd89259a8409f0bbf6`. This refresh integrates WoWSims master `41862b31e` and the nine class hotfix drafts. The selection file names every build in the new catalog. Across 40 builds, 1,116 matchups require 50,000 new iterations each. The other 1,368 retain their original simulation source, date and sampling error.

## Builds that need new results

- All six original Rogue builds use the revised talent layout and spell rows. Five additional builds use the draft allocations on their corresponding fork preset’s equipment and rotation. The changed Mutilate ranks, Shank effects and Death Mark cap also affect the comparison.
- Arms and Fury use the revised Warrior talent layout. Raging Blows leaves the tree, Improved Whirlwind uses its new ranks, and Reckless Swings uses its captured cost, global cooldown and proc conditions.
- Affliction Wrack uses the spell-power coefficient captured in the hotfix. Its numerical representation differs slightly from the fork’s earlier approximation. At eight iterations, every race and target type tested at 180 seconds differed from its previous result by less than 0.00001 DPS.

## Checks for the other 26 builds

The selection step compares each saved request, input fingerprint, gear and swaps, talents, rotation and setup with the new manifest. It requires the same legal races and all nine target and duration combinations. The source revision and the original 50,000-iteration measurements must exist in both published arenas.

- The item database adds Myrmidon’s Defender (8134) and Sparkleshell Mantle (13131). None of the 40 builds equips or swaps to them. Existing item rows, enchants and set membership are unchanged. The target scheduling changes apply to Dynamic Adds. The arenas use one continuously active target and end at a fixed time. Damage and execute behavior in those fights already used the previous target-death preview.
- **Hunter, Mage, Balance Druid and DPS Priest.** The seven Hunter presets already used the hotfixed shot and talent values. Mage already applied the updated Clearcasting rules, and its newly captured Missile Barrage change is tooltip text. Balance has no Cat energy or Poised interaction in these runs. The Priest change removes Binding Heal ranks. Neither DPS Priest rotation uses that heal.
- **Feral Cat, Feral Hybrid and Retribution.** The two Feral presets already applied Poised’s full-energy opener discount. The two Retribution presets already used the new Vengeance stack cap and the Physical and Holy weapon bonus. Eight-iteration comparisons reproduced every race on all three target types at 180 seconds for all four builds.
- **Elemental and Enhancement.** The 46 rendered Shaman spell rows and three talent curves changed by the original launch preview match their previous generated values. The new capture preserves the Lava Burst rank records. These presets still use Call of Thunder in that talent slot. Both builds reproduced the previous 180-second comparisons for all races and target types.
- **Other Warlocks.** Destruction, SM/Ruin and both Demonology builds retain their prior requests and do not channel Wrack. Shadow and Flame’s captured ranks give the same damage bonus as before. All four builds reproduced the previous 180-second comparisons across every race and target type.

The short runs check behavior alongside the code and spell-row comparisons. Reused rankings retain their original 50,000 iterations.
