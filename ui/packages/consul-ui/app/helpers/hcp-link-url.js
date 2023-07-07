import { helper } from '@ember/component/helper';

function substring([dc]) {
  const url = new URL('http://localhost:4200/services/consul/clusters/self-managed/link-existing');
  url.searchParams.append('dc', dc);
  url.searchParams.append('ui', `${window.location.host}/ui`);

  return url;
}

export default helper(substring);
