import semaphoreLogo from "../images/semaphore-logo-sign-black.svg";

const Navigation = () => {
  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-white shadow-sm">
      <div id="global-page-header" className="header flex items-center justify-between ph2 ph3-ns pv2">
        <a href="#" className="link flex items-center flex-shrink-0">
          <img src={semaphoreLogo} alt="Semaphore Logo" className="h-6" width={26} /> 
          <strong className="ml2 f3 black-90">SuperPlane</strong>
        </a>
        <div className="flex items-center flex-shrink-0">
          <div className="flex-shrink-0 pa1 ma1 pointer bg-animate hover-bg-washed-brown br-100 js-dropdown-color-trigger" data-template="profileMenu" aria-expanded="false">
            <span className="f7 db br-100 ba b--black-50" style={{ width: '24px', height: '24px' }}></span>    
          </div>
        </div>
      </div>
    </div>
  );
};

export default Navigation;